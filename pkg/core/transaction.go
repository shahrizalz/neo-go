package core

import (
	"fmt"
	"math/big"

	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/trigger"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// processTXInput processes single tx input.
func processTXInput(input *transaction.Input, unspent *state.UnspentCoin, block *block.Block, cache *cachedDao) error {
	if len(unspent.States) <= int(input.PrevIndex) {
		return fmt.Errorf("bad input: %s/%d", input.PrevHash.StringLE(), input.PrevIndex)
	}
	if unspent.States[input.PrevIndex].State&state.CoinSpent != 0 {
		return fmt.Errorf("double spend: %s/%d", input.PrevHash.StringLE(), input.PrevIndex)
	}
	unspent.States[input.PrevIndex].State |= state.CoinSpent
	unspent.States[input.PrevIndex].SpendHeight = block.Index
	prevTXOutput := &unspent.States[input.PrevIndex].Output
	account, err := cache.GetAccountStateOrNew(prevTXOutput.ScriptHash)
	if err != nil {
		return err
	}

	if prevTXOutput.AssetID.Equals(GoverningTokenID()) {
		err = account.Unclaimed.Put(&state.UnclaimedBalance{
			Tx:    input.PrevHash,
			Index: input.PrevIndex,
			Start: unspent.Height,
			End:   block.Index,
			Value: prevTXOutput.Amount,
		})
		if err != nil {
			return err
		}
		if err = processTXWithValidatorsSubtract(prevTXOutput, account, cache); err != nil {
			return err
		}
	}

	balancesLen := len(account.Balances[prevTXOutput.AssetID])
	if balancesLen <= 1 {
		delete(account.Balances, prevTXOutput.AssetID)
	} else {
		var index = -1
		for i, balance := range account.Balances[prevTXOutput.AssetID] {
			if balance.Tx.Equals(input.PrevHash) && balance.Index == input.PrevIndex {
				index = i
				break
			}
		}
		if index >= 0 {
			last := balancesLen - 1
			if last > index {
				account.Balances[prevTXOutput.AssetID][index] = account.Balances[prevTXOutput.AssetID][last]
			}
			account.Balances[prevTXOutput.AssetID] = account.Balances[prevTXOutput.AssetID][:last]
		}
	}
	return cache.PutAccountState(account)
}

// processInputs processes transaction inputs.
func processInputs(tx *transaction.Transaction, block *block.Block, cache *cachedDao) error {
	for _, inputs := range transaction.GroupInputsByPrevHash(tx.Inputs) {
		prevHash := inputs[0].PrevHash
		unspent, err := cache.GetUnspentCoinState(prevHash)
		if err != nil {
			return err
		}
		for _, input := range inputs {
			if err := processTXInput(input, unspent, block, cache); err != nil {
				return err
			}
		}
		if err = cache.PutUnspentCoinState(prevHash, unspent); err != nil {
			return err
		}
	}
	return nil
}

// processTXData process type-specific TX part.
func (bc *Blockchain) processTXData(tx *transaction.Transaction, block *block.Block, cache *cachedDao) error {
	switch t := tx.Data.(type) {
	case *transaction.RegisterTX:
		return cache.PutAssetState(&state.Asset{
			ID:         tx.Hash(),
			AssetType:  t.AssetType,
			Name:       t.Name,
			Amount:     t.Amount,
			Precision:  t.Precision,
			Owner:      t.Owner,
			Admin:      t.Admin,
			Expiration: bc.BlockHeight() + registeredAssetLifetime,
		})
	case *transaction.IssueTX:
		for _, res := range bc.GetTransactionResults(tx) {
			if res.Amount < 0 {
				asset, err := cache.GetAssetState(res.AssetID)
				if asset == nil || err != nil {
					return fmt.Errorf("issue failed: no asset %s or error %s", res.AssetID, err)
				}
				asset.Available -= res.Amount
				if err := cache.PutAssetState(asset); err != nil {
					return err
				}
			}
		}
	case *transaction.ClaimTX:
		return bc.processClaimTX(t, tx, block, cache)
	case *transaction.EnrollmentTX:
		return processEnrollmentTX(cache, t)
	case *transaction.StateTX:
		return processStateTX(cache, t)
	case *transaction.PublishTX:
		var properties smartcontract.PropertyState
		if t.NeedStorage {
			properties |= smartcontract.HasStorage
		}
		contract := &state.Contract{
			Script:      t.Script,
			ParamList:   t.ParamList,
			ReturnType:  t.ReturnType,
			Properties:  properties,
			Name:        t.Name,
			CodeVersion: t.CodeVersion,
			Author:      t.Author,
			Email:       t.Email,
			Description: t.Description,
		}
		return cache.PutContractState(contract)
	case *transaction.InvocationTX:
		return bc.processInvocationTX(t, tx, block, cache)
	}
	return nil
}

func (bc *Blockchain) processClaimTX(t *transaction.ClaimTX, tx *transaction.Transaction, block *block.Block, cache *cachedDao) error {
	// Remove claimed NEO from spent coins making it unavalaible for
	// additional claims.
	for _, input := range t.Claims {
		scs, err := cache.GetUnspentCoinState(input.PrevHash)
		if err == nil {
			if len(scs.States) <= int(input.PrevIndex) {
				err = errors.New("invalid claim index")
			} else if scs.States[input.PrevIndex].State&state.CoinClaimed != 0 {
				err = errors.New("double claim")
			}
		}
		if err != nil {
			// We can't really do anything about it
			// as it's a transaction in a signed block.
			bc.log.Warn("FALSE OR DOUBLE CLAIM",
				zap.String("PrevHash", input.PrevHash.StringLE()),
				zap.Uint16("PrevIndex", input.PrevIndex),
				zap.String("tx", tx.Hash().StringLE()),
				zap.Uint32("block", block.Index),
			)
			// "Strict" mode.
			if bc.config.VerifyTransactions {
				return err
			}
			break
		}

		acc, err := cache.GetAccountState(scs.States[input.PrevIndex].ScriptHash)
		if err != nil {
			return err
		}

		scs.States[input.PrevIndex].State |= state.CoinClaimed
		if err = cache.PutUnspentCoinState(input.PrevHash, scs); err != nil {
			return err
		}

		changed := acc.Unclaimed.Remove(input.PrevHash, input.PrevIndex)
		if !changed {
			bc.log.Warn("no spent coin in the account",
				zap.String("tx", tx.Hash().StringLE()),
				zap.String("input", input.PrevHash.StringLE()),
				zap.String("account", acc.ScriptHash.String()))
		} else if err := cache.PutAccountState(acc); err != nil {
			return err
		}
	}
	return nil
}

func (bc *Blockchain) processInvocationTX(t *transaction.InvocationTX, tx *transaction.Transaction, block *block.Block, cache *cachedDao) error {
	systemInterop := bc.newInteropContext(trigger.Application, cache.store, block, tx)
	v := bc.spawnVMWithInterops(systemInterop)
	v.SetCheckedHash(tx.VerificationHash().BytesBE())
	v.LoadScript(t.Script)
	v.SetPriceGetter(getPrice)
	if bc.config.FreeGasLimit > 0 {
		v.SetGasLimit(bc.config.FreeGasLimit + t.Gas)
	}

	err := v.Run()
	if !v.HasFailed() {
		_, err := systemInterop.dao.Persist()
		if err != nil {
			return errors.Wrap(err, "failed to persist invocation results")
		}
		for _, note := range systemInterop.notifications {
			arr, ok := note.Item.Value().([]vm.StackItem)
			if !ok || len(arr) != 4 {
				continue
			}
			op, ok := arr[0].Value().([]byte)
			if !ok || string(op) != "transfer" {
				continue
			}
			from, ok := arr[1].Value().([]byte)
			if !ok {
				continue
			}
			to, ok := arr[2].Value().([]byte)
			if !ok {
				continue
			}
			amount, ok := arr[3].Value().(*big.Int)
			if !ok {
				bs, ok := arr[3].Value().([]byte)
				if !ok {
					continue
				}
				amount = emit.BytesToInt(bs)
			}
			bc.processNEP5Transfer(cache, tx, block, note.ScriptHash, from, to, amount.Int64())
		}
	} else {
		bc.log.Warn("contract invocation failed",
			zap.String("tx", tx.Hash().StringLE()),
			zap.Uint32("block", block.Index),
			zap.Error(err))
	}
	aer := &state.AppExecResult{
		TxHash:      tx.Hash(),
		Trigger:     trigger.Application,
		VMState:     v.State(),
		GasConsumed: v.GasConsumed(),
		Stack:       v.Estack().ToContractParameters(),
		Events:      systemInterop.notifications,
	}
	err = cache.PutAppExecResult(aer)
	return errors.Wrap(err, "failed to store notifications")
}
