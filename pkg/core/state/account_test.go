package state

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/internal/random"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestDecodeEncodeAccountState(t *testing.T) {
	var (
		n        = 10
		balances = make(map[util.Uint256][]UnspentBalance)
		votes    = make([]*keys.PublicKey, n)
	)
	for i := 0; i < n; i++ {
		asset := random.Uint256()
		for j := 0; j < i+1; j++ {
			balances[asset] = append(balances[asset], UnspentBalance{
				Tx:    random.Uint256(),
				Index: uint16(random.Int(0, 65535)),
				Value: util.Fixed8(int64(random.Int(1, 10000))),
			})
		}
		k, err := keys.NewPrivateKey()
		assert.Nil(t, err)
		votes[i] = k.PublicKey()
	}

	a := &Account{
		Version:    0,
		ScriptHash: random.Uint160(),
		IsFrozen:   true,
		Votes:      votes,
		Balances:   balances,
	}

	buf := io.NewBufBinWriter()
	a.EncodeBinary(buf.BinWriter)
	assert.Nil(t, buf.Err)

	aDecode := &Account{}
	r := io.NewBinReaderFromBuf(buf.Bytes())
	aDecode.DecodeBinary(r)
	assert.Nil(t, r.Err)

	assert.Equal(t, a.Version, aDecode.Version)
	assert.Equal(t, a.ScriptHash, aDecode.ScriptHash)
	assert.Equal(t, a.IsFrozen, aDecode.IsFrozen)

	for i, vote := range a.Votes {
		assert.Equal(t, vote.X, aDecode.Votes[i].X)
	}
	assert.Equal(t, a.Balances, aDecode.Balances)
}

func TestAccountStateBalanceValues(t *testing.T) {
	asset1 := random.Uint256()
	asset2 := random.Uint256()
	as := Account{Balances: make(map[util.Uint256][]UnspentBalance)}
	ref := 0
	for i := 0; i < 10; i++ {
		ref += i
		as.Balances[asset1] = append(as.Balances[asset1], UnspentBalance{Value: util.Fixed8(i)})
		as.Balances[asset2] = append(as.Balances[asset2], UnspentBalance{Value: util.Fixed8(i * 10)})
	}
	bVals := as.GetBalanceValues()
	assert.Equal(t, util.Fixed8(ref), bVals[asset1])
	assert.Equal(t, util.Fixed8(ref*10), bVals[asset2])
}

func BenchmarkAccount_EncodeBinary(b *testing.B) {
	acc := randomAccount(10)

	b.Run("EncodeBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = encodeIO(acc)
		}
	})

	b.Run("gob.Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = encodeGOB(acc)
		}
	})

	buf := encodeIO(acc)
	b.Run("DecodeBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = decodeIO(buf)
		}
	})

	buf = encodeGOB(acc)
	b.Run("gob.Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = decodeGOB(buf)
		}
	})
}

func encodeIO(a *Account) []byte {
	buf := io.NewBufBinWriter()
	a.EncodeBinary(buf.BinWriter)
	if buf.Err != nil {
		panic(buf.Err)
	}
	return buf.Bytes()
}

func decodeIO(buf []byte) *Account {
	var a Account
	r := io.NewBinReaderFromBuf(buf)
	a.DecodeBinary(r)
	if r.Err != nil {
		panic(r.Err)
	}
	return &a
}

func encodeGOB(a *Account) []byte {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(a); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func decodeGOB(buf []byte) *Account {
	var a Account
	r := bytes.NewReader(buf)
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&a); err != nil {
		panic(err)
	}
	return &a
}

func randomAccount(n int) *Account {
	balances := make(map[util.Uint256][]UnspentBalance)
	votes := make([]*keys.PublicKey, n)
	unclaimed := UnclaimedBalances{}

	for i := 0; i < n; i++ {
		asset := random.Uint256()
		for j := 0; j < i+1; j++ {
			balances[asset] = append(balances[asset], UnspentBalance{
				Tx:    random.Uint256(),
				Index: uint16(random.Int(0, 65535)),
				Value: util.Fixed8(int64(random.Int(1, 10000))),
			})
		}
		k, err := keys.NewPrivateKey()
		if err != nil {
			panic(err)
		}
		votes[i] = k.PublicKey()

		err = unclaimed.Put(&UnclaimedBalance{
			Tx:    random.Uint256(),
			Index: uint16(rand.Uint32()),
			Start: rand.Uint32(),
			End:   rand.Uint32(),
			Value: util.Fixed8(rand.Int63()),
		})
		if err != nil {
			panic(err)
		}
	}

	return &Account{
		ScriptHash: random.Uint160(),
		IsFrozen:   true,
		Votes:      votes,
		Balances:   balances,
		Unclaimed:  unclaimed,
	}
}
