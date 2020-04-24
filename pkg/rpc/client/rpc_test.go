package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/rpc/request"
	"github.com/nspcc-dev/neo-go/pkg/rpc/response/result"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rpcClientTestCase struct {
	name           string
	invoke         func(c *Client) (interface{}, error)
	serverResponse string
	result         func(c *Client) interface{}
	check          func(t *testing.T, c *Client, result interface{})
}

// rpcClientTestCases contains `serverResponse` json data fetched from examples
// published in official C# JSON-RPC API v2.10.3 reference
// (see https://docs.neo.org/docs/en-us/reference/rpc/latest-version/api.html)
var rpcClientTestCases = map[string][]rpcClientTestCase{
	"getaccountstate": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetAccountState("")
			},
			serverResponse: `{"jsonrpc":"2.0","id": 1,"result":{"version":0,"script_hash":"0x1179716da2e9523d153a35fb3ad10c561b1e5b1a","frozen":false,"votes":[],"balances":[{"asset":"0x7a37715546c6cfa5bac8d7f7e87c667a1e5a6ba0601238be475ab8c79a5abcf5","value":"94"}]}}`,
			result: func(c *Client) interface{} {
				scriptHash, err := util.Uint160DecodeStringLE("1179716da2e9523d153a35fb3ad10c561b1e5b1a")
				if err != nil {
					panic(err)
				}
				return &result.AccountState{
					Version:    0,
					ScriptHash: scriptHash,
					IsFrozen:   false,
					Votes:      []*keys.PublicKey{},
					Balances: result.Balances{
						result.Balance{
							Asset: core.GoverningTokenID(),
							Value: util.Fixed8FromInt64(94),
						},
					},
				}
			},
		},
	},
	"getapplicationlog": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetApplicationLog(util.Uint256{})
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"txid":"0x17145a039fca704fcdbeb46e6b210af98a1a9e5b9768e46ffc38f71c79ac2521","executions":[{"trigger":"Application","contract":"0xb9fa3b421eb749d5dd585fe1c1133b311a14bcb1","vmstate":"HALT","gas_consumed":"1","stack":[{"type":"Integer","value":1}],"notifications":[]}]}}`,
			result: func(c *Client) interface{} {
				txHash, err := util.Uint256DecodeStringLE("17145a039fca704fcdbeb46e6b210af98a1a9e5b9768e46ffc38f71c79ac2521")
				if err != nil {
					panic(err)
				}
				scriptHash, err := util.Uint160DecodeStringLE("b9fa3b421eb749d5dd585fe1c1133b311a14bcb1")
				if err != nil {
					panic(err)
				}
				return &result.ApplicationLog{
					TxHash: txHash,
					Executions: []result.Execution{
						{
							Trigger:     "Application",
							ScriptHash:  scriptHash,
							VMState:     "HALT",
							GasConsumed: util.Fixed8FromInt64(1),
							Stack:       []smartcontract.Parameter{{Type: smartcontract.IntegerType, Value: int64(1)}},
							Events:      []result.NotificationEvent{},
						},
					},
				}
			},
		},
	},
	"getassetstate": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetAssetState(util.Uint256{})
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"id":"0x7a37715546c6cfa5bac8d7f7e87c667a1e5a6ba0601238be475ab8c79a5abcf5","type":0,"name":"NEO","amount":"100000000","available":"100000000","precision":0,"owner":"00","admin":"Abf2qMs1pzQb8kYk9RuxtUb9jtRKJVuBJt","issuer":"AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM","expiration":4000000,"is_frozen":false}}`,
			result: func(c *Client) interface{} {
				return &result.AssetState{
					ID:         core.GoverningTokenID(),
					AssetType:  0,
					Name:       "NEO",
					Amount:     util.Fixed8FromInt64(100000000),
					Available:  util.Fixed8FromInt64(100000000),
					Precision:  0,
					Owner:      "00",
					Admin:      "Abf2qMs1pzQb8kYk9RuxtUb9jtRKJVuBJt",
					Issuer:     "AFmseVrdL9f9oyCzZefL9tG6UbvhPbdYzM",
					Expiration: 4000000,
					IsFrozen:   false,
				}
			},
		},
	},
	"getbestblockhash": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBestBlockHash()
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":"0x773dd2dae4a9c9275290f89b56e67d7363ea4826dfd4fc13cc01cf73a44b0d0e"}`,
			result: func(c *Client) interface{} {
				result, err := util.Uint256DecodeStringLE("773dd2dae4a9c9275290f89b56e67d7363ea4826dfd4fc13cc01cf73a44b0d0e")
				if err != nil {
					panic(err)
				}
				return result
			},
		},
	},
	"getblock": {
		{
			name: "byIndex_positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndex(5)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":"0000000062af8cd56d179044215cc8611f75bd96d896f1026c5b42994ae7707df8d82bd3c9f774f449fec7135b506faffaaeee603e2b82e01dec7d0f706789aa1bb983ae0ec7a25e0000000005000000e903736ceceeceae1806eee0e3ec61e7cce476ce01fd08010c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c7763594130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb020057040000000000000000d5040000e903736ceceeceae1806eee0e3ec61e7cce476ce0500000000000001fd08010c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce94130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"}`,
			result:         func(c *Client) interface{} { return &block.Block{} },
			check: func(t *testing.T, c *Client, result interface{}) {
				res, ok := result.(*block.Block)
				require.True(t, ok)
				assert.Equal(t, uint32(0), res.Version)
				assert.Equal(t, "424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9", res.Hash().StringLE())
				assert.Equal(t, "d32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62", res.PrevHash.StringLE())
				assert.Equal(t, "ae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9", res.MerkleRoot.StringLE())
				assert.Equal(t, 1, len(res.Transactions))
				assert.Equal(t, "ae63e96d984673b038c83cfcb94323e37bdab29a53921823544b50df9f7edb54", res.Transactions[0].Hash().StringLE())
			},
		},
		{
			name: "byIndex_verbose_positive",
			invoke: func(c *Client) (i interface{}, err error) {
				return c.GetBlockByIndexVerbose(5)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"hash":"0x424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9","size":977,"version":0,"nextblockhash":"0xc2ce96d861414ad229101cc9afaec4ae500f730a2180b54bd14a8dd6147bc8c3","previousblockhash":"0xd32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62","merkleroot":"0xae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9","time":1587726094,"index":5,"consensus_data":{"primary":0,"nonce":"0000000000000457"},"nextconsensus":"Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy","confirmations":203,"script":{"invocation":"0c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c77635","verification":"130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"},"tx":[{"sys_fee":"0","net_fee":"0","txid":"0xae63e96d984673b038c83cfcb94323e37bdab29a53921823544b50df9f7edb54","size":450,"type":"MinerTransaction","version":0,"nonce":1237,"sender":"Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy","valid_until_block":5,"attributes":[],"vin":[],"vout":[],"scripts":[{"invocation":"0c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce","verification":"130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"}]}]}}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint256DecodeStringLE("424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9")
				if err != nil {
					panic(err)
				}
				nextBlockHash, err := util.Uint256DecodeStringLE("c2ce96d861414ad229101cc9afaec4ae500f730a2180b54bd14a8dd6147bc8c3")
				if err != nil {
					panic(err)
				}
				prevBlockHash, err := util.Uint256DecodeStringLE("d32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62")
				if err != nil {
					panic(err)
				}
				merkleRoot, err := util.Uint256DecodeStringLE("ae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9")
				if err != nil {
					panic(err)
				}
				invScript, err := hex.DecodeString("0c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c77635")
				if err != nil {
					panic(err)
				}
				verifScript, err := hex.DecodeString("130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb")
				if err != nil {
					panic(err)
				}
				sender, err := address.StringToUint160("Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy")
				if err != nil {
					panic(err)
				}
				txInvScript, err := hex.DecodeString("0c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce")
				if err != nil {
					panic(err)
				}
				txVerifScript, err := hex.DecodeString("130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb")
				if err != nil {
					panic(err)
				}
				tx := transaction.NewMinerTXWithNonce(1237)
				tx.ValidUntilBlock = 5
				tx.Sender = sender
				tx.Scripts = []transaction.Witness{
					{
						InvocationScript:   txInvScript,
						VerificationScript: txVerifScript,
					},
				}
				// Update hashes for correct result comparison.
				_ = tx.Hash()
				return &result.Block{
					Hash:              hash,
					Size:              977,
					Version:           0,
					NextBlockHash:     &nextBlockHash,
					PreviousBlockHash: prevBlockHash,
					MerkleRoot:        merkleRoot,
					Time:              1587726094,
					Index:             5,
					NextConsensus:     "Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy",
					Confirmations:     203,
					ConsensusData: result.ConsensusData{
						PrimaryIndex: 0,
						Nonce:        "0000000000000457",
					},
					Script: transaction.Witness{
						InvocationScript:   invScript,
						VerificationScript: verifScript,
					},
					Tx: []result.Tx{{
						Transaction: tx,
						Fees: result.Fees{
							SysFee: 0,
							NetFee: 0,
						},
					}},
				}
			},
		},
		{
			name: "byHash_positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("e9f71b58764157f1e2c3e29f217e654b57956ee7c8a60496b03ea85e39084b42")
				if err != nil {
					panic(err)
				}
				return c.GetBlockByHash(hash)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":"0000000062af8cd56d179044215cc8611f75bd96d896f1026c5b42994ae7707df8d82bd3c9f774f449fec7135b506faffaaeee603e2b82e01dec7d0f706789aa1bb983ae0ec7a25e0000000005000000e903736ceceeceae1806eee0e3ec61e7cce476ce01fd08010c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c7763594130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb020057040000000000000000d5040000e903736ceceeceae1806eee0e3ec61e7cce476ce0500000000000001fd08010c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce94130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"}`,
			result:         func(c *Client) interface{} { return &block.Block{} },
			check: func(t *testing.T, c *Client, result interface{}) {
				res, ok := result.(*block.Block)
				require.True(t, ok)
				assert.Equal(t, uint32(0), res.Version)
				assert.Equal(t, "424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9", res.Hash().StringLE())
				assert.Equal(t, "d32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62", res.PrevHash.StringLE())
				assert.Equal(t, "ae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9", res.MerkleRoot.StringLE())
				assert.Equal(t, 1, len(res.Transactions))
				assert.Equal(t, "ae63e96d984673b038c83cfcb94323e37bdab29a53921823544b50df9f7edb54", res.Transactions[0].Hash().StringLE())
			},
		},
		{
			name: "byHash_verbose_positive",
			invoke: func(c *Client) (i interface{}, err error) {
				hash, err := util.Uint256DecodeStringLE("e9f71b58764157f1e2c3e29f217e654b57956ee7c8a60496b03ea85e39084b42")
				if err != nil {
					panic(err)
				}
				return c.GetBlockByHashVerbose(hash)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"hash":"0x424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9","size":977,"version":0,"nextblockhash":"0xc2ce96d861414ad229101cc9afaec4ae500f730a2180b54bd14a8dd6147bc8c3","previousblockhash":"0xd32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62","merkleroot":"0xae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9","time":1587726094,"index":5,"consensus_data":{"primary":0,"nonce":"0000000000000457"},"nextconsensus":"Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy","confirmations":203,"script":{"invocation":"0c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c77635","verification":"130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"},"tx":[{"sys_fee":"0","net_fee":"0","txid":"0xae63e96d984673b038c83cfcb94323e37bdab29a53921823544b50df9f7edb54","size":450,"type":"MinerTransaction","version":0,"nonce":1237,"sender":"Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy","valid_until_block":5,"attributes":[],"vin":[],"vout":[],"scripts":[{"invocation":"0c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce","verification":"130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb"}]}]}}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint256DecodeStringLE("424b08395ea83eb09604a6c8e76e95574b657e219fe2c3e2f1574176581bf7e9")
				if err != nil {
					panic(err)
				}
				nextBlockHash, err := util.Uint256DecodeStringLE("c2ce96d861414ad229101cc9afaec4ae500f730a2180b54bd14a8dd6147bc8c3")
				if err != nil {
					panic(err)
				}
				prevBlockHash, err := util.Uint256DecodeStringLE("d32bd8f87d70e74a99425b6c02f196d896bd751f61c85c214490176dd58caf62")
				if err != nil {
					panic(err)
				}
				merkleRoot, err := util.Uint256DecodeStringLE("ae83b91baa8967700f7dec1de0822b3e60eeaefaaf6f505b13c7fe49f474f7c9")
				if err != nil {
					panic(err)
				}
				invScript, err := hex.DecodeString("0c408e48ace06fdd7d9bf536b6cb683f7edd336c60a707df8110f69121273fe7e0353e574c55abf2961ac4f7f2bfef44af07e6121f42e5e2115517b29060e3a7dd3e0c40d56609addaa61f06d9df159f7008ffb889d605742baaf7f95a8283469d6e5a4a76c5814f24efa0452e3c6723d88e43833e917551808d05aca8d46a17f25c72440c40fa0b66a2a41933e39685f7cbf45ba0cef286b3eed5f7d1cb60db4bac3a9c55212efb5b1f4a4c5512b2562f8e0a2ebfbc8951734ca53243ec963bd6839773f5910c40c1c0de79304d8ad7e204dceb880325694e5c34abb25ff23beb61e931ecf384e4f06c13a5ea56273c400ecac9408a3eb8e8cf3b0b358f7b2b6ac5120bb5c77635")
				if err != nil {
					panic(err)
				}
				verifScript, err := hex.DecodeString("130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb")
				if err != nil {
					panic(err)
				}
				sender, err := address.StringToUint160("Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy")
				if err != nil {
					panic(err)
				}
				txInvScript, err := hex.DecodeString("0c40a9c72069ad0365a8f0787a236ec60293a9846172e9cdeaeb665586d6c72545bcfa694422f8ccd3e76bce7e27ac8099cc9b3f6322bcfeaf971c9b481a1a308a350c4048f7c2a176a7c8eb73f881aacb0a5bc52bb3b2eeeb2341031496aaadbc043dda02d8c79935ac27ecda0dc7c2561af056946e82ff1a819b56461ad32fce83ab960c4036a238579bbe505150f2ea2e4172eb83cfd614af00c1cfe36791a1eb12cb5565f37668fa09a0fcb2528fffe377c96ec9d63d18aa19a5d6c24c5c97034d1811250c4007c3826543bc03b3b6cecf48fb30ff24033c1aad7a946ac6c54e7fa90173ff3b0fe181936079fc0e7030bdde2b655ae3a7101b8a0bd85fc98de83bb72739a9ce")
				if err != nil {
					panic(err)
				}
				txVerifScript, err := hex.DecodeString("130c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e0c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd620c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc20c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699140b683073b3bb")
				if err != nil {
					panic(err)
				}
				tx := transaction.NewMinerTXWithNonce(1237)
				tx.ValidUntilBlock = 5
				tx.Sender = sender
				tx.Scripts = []transaction.Witness{
					{
						InvocationScript:   txInvScript,
						VerificationScript: txVerifScript,
					},
				}
				// Update hashes for correct result comparison.
				_ = tx.Hash()
				return &result.Block{
					Hash:              hash,
					Size:              977,
					Version:           0,
					NextBlockHash:     &nextBlockHash,
					PreviousBlockHash: prevBlockHash,
					MerkleRoot:        merkleRoot,
					Time:              1587726094,
					Index:             5,
					NextConsensus:     "Ad1wDxzcRiRSryvJobNV211Tv7UUiziPXy",
					Confirmations:     203,
					ConsensusData: result.ConsensusData{
						PrimaryIndex: 0,
						Nonce:        "0000000000000457",
					},
					Script: transaction.Witness{
						InvocationScript:   invScript,
						VerificationScript: verifScript,
					},
					Tx: []result.Tx{{
						Transaction: tx,
						Fees: result.Fees{
							SysFee: 0,
							NetFee: 0,
						},
					}},
				}
			},
		},
	},
	"getblockcount": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockCount()
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":991991}`,
			result: func(c *Client) interface{} {
				return uint32(991991)
			},
		},
	},
	"getblockhash": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHash(1)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":"0x4c1e879872344349067c3b1a30781eeb4f9040d3795db7922f513f6f9660b9b2"}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint256DecodeStringLE("4c1e879872344349067c3b1a30781eeb4f9040d3795db7922f513f6f9660b9b2")
				if err != nil {
					panic(err)
				}
				return hash
			},
		},
	},
	"getblockheader": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("68e4bd688b852e807eef13a0ff7da7b02223e359a35153667e88f9cb4a3b0801")
				if err != nil {
					panic(err)
				}
				return c.GetBlockHeader(hash)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":"00000000d039da5e49d63eb0533437d24ff8ceb6aeacf88680599c39f0ffca8948dfcdb94a3def1fca91cf45d69358414e3be77f7621e557f4cebbdb79a47d3cf56ac007f920a05e0000000001000000d60ac443bb800fb08261e75fa5925d747d48586101fd04014055041db6a59c99ab98137cc57e1e56a0a89856a311b2d2fc0aec76ec714c7616edc8fc5c9b81b27f25b7db1a61f64be0730a9cc103efcea1195cc3fe55843e264027e49c647f48bb08d3c32b79ee3432005ea577d7e497f78b46f1e81858848f961b557fb42a92e8eb4433fed203c917cbebb2138a31ed86750fb769d1e70956c0404c20054aa8bd45b520cba9410a9dd6c256481066bb657d7793fbba5551898c91b6dde81285fac841753ccfdd3193d08f19d5431313fa0d926ca965072a5fa3384026b0705078409bcc62fb98bb985edc387edeaaeba37bb7642d88a90762b2c2a62d9b61d53c097d548a368e450c4d995a178d5af28d4c93698233c52de05e3f0094534c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e4c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd624c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc24c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee6995450683073b3bb00"}`,
			result:         func(c *Client) interface{} { return &block.Header{} },
			check: func(t *testing.T, c *Client, result interface{}) {
				res, ok := result.(*block.Header)
				require.True(t, ok)
				assert.Equal(t, uint32(0), res.Version)
				assert.Equal(t, "68e4bd688b852e807eef13a0ff7da7b02223e359a35153667e88f9cb4a3b0801", res.Hash().StringLE())
				assert.Equal(t, "b9cddf4889cafff0399c598086f8acaeb6cef84fd2373453b03ed6495eda39d0", res.PrevHash.StringLE())
				assert.Equal(t, "07c06af53c7da479dbbbcef457e521767fe73b4e415893d645cf91ca1fef3d4a", res.MerkleRoot.StringLE())
			},
		},
		{
			name: "verbose_positive",
			invoke: func(c *Client) (i interface{}, err error) {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				return c.GetBlockHeaderVerbose(hash)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"hash":"0xe93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c","size":442,"version":0,"previousblockhash":"0x996e37358dc369912041f966f8c5d8d3a8255ba5dcbd3447f8a82b55db869099","merkleroot":"0xcb6ddb5f99d6af4c94a6c396d5294472f2eebc91a2c933e0f527422296fa9fb2","time":1541215200,"index":1,"nonce":"51b484a2fe49ed4d","nextconsensus":"AZ81H31DMWzbSnFDLFkzh9vHwaDLayV7fU","script":{"invocation":"40356a91d94e398170e47447d6a0f60aa5470e209782a5452403115a49166db3e1c4a3898122db19f779c30f8ccd0b7d401acdf71eda340655e4ae5237a64961bf4034dd47955e5a71627dafc39dd92999140e9eaeec6b11dbb2b313efa3f1093ed915b4455e199c69ec53778f94ffc236b92f8b97fff97a1f6bbb3770c0c0b3844a40fbe743bd5c90b2f5255e0b073281d7aeb2fb516572f36bec8446bcc37ac755cbf10d08b16c95644db1b2dddc2df5daa377880b20198fc7b967ac6e76474b22df","verification":"532102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd622102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc22103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee69954ae"},"confirmations":20061,"nextblockhash":"0xcc37d5bc460e72c9423015cb8d579c13e7b03b93bfaa1a23cf4fa777988e035f"}}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				nextBlockHash, err := util.Uint256DecodeStringLE("cc37d5bc460e72c9423015cb8d579c13e7b03b93bfaa1a23cf4fa777988e035f")
				if err != nil {
					panic(err)
				}
				prevBlockHash, err := util.Uint256DecodeStringLE("996e37358dc369912041f966f8c5d8d3a8255ba5dcbd3447f8a82b55db869099")
				if err != nil {
					panic(err)
				}
				merkleRoot, err := util.Uint256DecodeStringLE("cb6ddb5f99d6af4c94a6c396d5294472f2eebc91a2c933e0f527422296fa9fb2")
				if err != nil {
					panic(err)
				}
				invScript, err := hex.DecodeString("40356a91d94e398170e47447d6a0f60aa5470e209782a5452403115a49166db3e1c4a3898122db19f779c30f8ccd0b7d401acdf71eda340655e4ae5237a64961bf4034dd47955e5a71627dafc39dd92999140e9eaeec6b11dbb2b313efa3f1093ed915b4455e199c69ec53778f94ffc236b92f8b97fff97a1f6bbb3770c0c0b3844a40fbe743bd5c90b2f5255e0b073281d7aeb2fb516572f36bec8446bcc37ac755cbf10d08b16c95644db1b2dddc2df5daa377880b20198fc7b967ac6e76474b22df")
				if err != nil {
					panic(err)
				}
				verifScript, err := hex.DecodeString("532102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd622102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc22103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee69954ae")
				if err != nil {
					panic(err)
				}
				return &result.Header{
					Hash:          hash,
					Size:          442,
					Version:       0,
					NextBlockHash: &nextBlockHash,
					PrevBlockHash: prevBlockHash,
					MerkleRoot:    merkleRoot,
					Timestamp:     1541215200,
					Index:         1,
					NextConsensus: "AZ81H31DMWzbSnFDLFkzh9vHwaDLayV7fU",
					Confirmations: 20061,
					Script: transaction.Witness{
						InvocationScript:   invScript,
						VerificationScript: verifScript,
					},
				}
			},
		},
	},
	"getblocksysfee": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockSysFee(1)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":"195500"}`,
			result: func(c *Client) interface{} {
				return util.Fixed8FromInt64(195500)
			},
		},
	},
	"getclaimable": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetClaimable("AGofsxAUDwt52KjaB664GYsqVAkULYvKNt")
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"claimable":[{"txid":"52ba70ef18e879785572c917795cd81422c3820b8cf44c24846a30ee7376fd77","n":1,"value":800000,"start_height":476496,"end_height":488154,"generated":746.112,"sys_fee": 3.92,"unclaimed":750.032}],"address":"AGofsxAUDwt52KjaB664GYsqVAkULYvKNt","unclaimed": 750.032}}`,
			result: func(c *Client) interface{} {
				txID, err := util.Uint256DecodeStringLE("52ba70ef18e879785572c917795cd81422c3820b8cf44c24846a30ee7376fd77")
				if err != nil {
					panic(err)
				}
				return &result.ClaimableInfo{
					Spents: []result.Claimable{
						{
							Tx:          txID,
							N:           1,
							Value:       util.Fixed8FromInt64(800000),
							StartHeight: 476496,
							EndHeight:   488154,
							Generated:   util.Fixed8FromFloat(746.112),
							SysFee:      util.Fixed8FromFloat(3.92),
							Unclaimed:   util.Fixed8FromFloat(750.032),
						}},
					Address:   "AGofsxAUDwt52KjaB664GYsqVAkULYvKNt",
					Unclaimed: util.Fixed8FromFloat(750.032),
				}
			},
		},
	},
	"getconnectioncount": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetConnectionCount()
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":10}`,
			result: func(c *Client) interface{} {
				return 10
			},
		},
	},
	"getcontractstate": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint160DecodeStringLE("dc675afc61a7c0f7b3d2682bf6e1d8ed865a0e5f")
				if err != nil {
					panic(err)
				}
				return c.GetContractState(hash)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"version":0,"hash":"0xdc675afc61a7c0f7b3d2682bf6e1d8ed865a0e5f","script":"5fc56b6c766b00527ac46c766b51527ac46107576f6f6c6f6e676c766b52527ac403574e476c766b53527ac4006c766b54527ac4210354ae498221046c666efebbaee9bd0eb4823469c98e748494a92a71f346b1a6616c766b55527ac46c766b00c3066465706c6f79876c766b56527ac46c766b56c36416006c766b55c36165f2026c766b57527ac462d8016c766b55c36165d801616c766b00c30b746f74616c537570706c79876c766b58527ac46c766b58c36440006168164e656f2e53746f726167652e476574436f6e7465787406737570706c79617c680f4e656f2e53746f726167652e4765746c766b57527ac46270016c766b00c3046e616d65876c766b59527ac46c766b59c36412006c766b52c36c766b57527ac46247016c766b00c30673796d626f6c876c766b5a527ac46c766b5ac36412006c766b53c36c766b57527ac4621c016c766b00c308646563696d616c73876c766b5b527ac46c766b5bc36412006c766b54c36c766b57527ac462ef006c766b00c30962616c616e63654f66876c766b5c527ac46c766b5cc36440006168164e656f2e53746f726167652e476574436f6e746578746c766b51c351c3617c680f4e656f2e53746f726167652e4765746c766b57527ac46293006c766b51c300c36168184e656f2e52756e74696d652e436865636b5769746e657373009c6c766b5d527ac46c766b5dc3640e00006c766b57527ac46255006c766b00c3087472616e73666572876c766b5e527ac46c766b5ec3642c006c766b51c300c36c766b51c351c36c766b51c352c36165d40361527265c9016c766b57527ac4620e00006c766b57527ac46203006c766b57c3616c756653c56b6c766b00527ac4616168164e656f2e53746f726167652e476574436f6e746578746c766b00c3617c680f4e656f2e53746f726167652e4765746165700351936c766b51527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b51c361651103615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e7465787406737570706c79617c680f4e656f2e53746f726167652e4765746165f40251936c766b52527ac46168164e656f2e53746f726167652e476574436f6e7465787406737570706c796c766b52c361659302615272680f4e656f2e53746f726167652e50757461616c756653c56b6c766b00527ac461516c766b51527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b51c361654002615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e7465787406737570706c796c766b51c361650202615272680f4e656f2e53746f726167652e50757461516c766b52527ac46203006c766b52c3616c756659c56b6c766b00527ac46c766b51527ac46c766b52527ac4616168164e656f2e53746f726167652e476574436f6e746578746c766b00c3617c680f4e656f2e53746f726167652e4765746c766b53527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b51c3617c680f4e656f2e53746f726167652e4765746c766b54527ac46c766b53c3616576016c766b52c3946c766b55527ac46c766b54c3616560016c766b52c3936c766b56527ac46c766b55c300a2640d006c766b52c300a2620400006c766b57527ac46c766b57c364ec00616168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b55c36165d800615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e746578746c766b51c36c766b56c361659c00615272680f4e656f2e53746f726167652e5075746155c57600135472616e73666572205375636365737366756cc476516c766b00c3c476526c766b51c3c476536c766b52c3c476546168184e656f2e426c6f636b636861696e2e476574486569676874c46168124e656f2e52756e74696d652e4e6f7469667961516c766b58527ac4620e00006c766b58527ac46203006c766b58c3616c756653c56b6c766b00527ac4616c766b00c36c766b51527ac46c766b51c36c766b52527ac46203006c766b52c3616c756653c56b6c766b00527ac461516c766b00c36a527a527ac46c766b51c36c766b52527ac46203006c766b52c3616c7566","parameters":["ByteArray"],"returntype":"ByteArray","name":"Woolong","code_version":"0.9.2","author":"lllwvlvwlll","email":"lllwvlvwlll@gmail.com","description":"GO NEO!!!","properties":{"storage":true,"dynamic_invoke":false}}}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint160DecodeStringLE("dc675afc61a7c0f7b3d2682bf6e1d8ed865a0e5f")
				if err != nil {
					panic(err)
				}
				script, err := hex.DecodeString("5fc56b6c766b00527ac46c766b51527ac46107576f6f6c6f6e676c766b52527ac403574e476c766b53527ac4006c766b54527ac4210354ae498221046c666efebbaee9bd0eb4823469c98e748494a92a71f346b1a6616c766b55527ac46c766b00c3066465706c6f79876c766b56527ac46c766b56c36416006c766b55c36165f2026c766b57527ac462d8016c766b55c36165d801616c766b00c30b746f74616c537570706c79876c766b58527ac46c766b58c36440006168164e656f2e53746f726167652e476574436f6e7465787406737570706c79617c680f4e656f2e53746f726167652e4765746c766b57527ac46270016c766b00c3046e616d65876c766b59527ac46c766b59c36412006c766b52c36c766b57527ac46247016c766b00c30673796d626f6c876c766b5a527ac46c766b5ac36412006c766b53c36c766b57527ac4621c016c766b00c308646563696d616c73876c766b5b527ac46c766b5bc36412006c766b54c36c766b57527ac462ef006c766b00c30962616c616e63654f66876c766b5c527ac46c766b5cc36440006168164e656f2e53746f726167652e476574436f6e746578746c766b51c351c3617c680f4e656f2e53746f726167652e4765746c766b57527ac46293006c766b51c300c36168184e656f2e52756e74696d652e436865636b5769746e657373009c6c766b5d527ac46c766b5dc3640e00006c766b57527ac46255006c766b00c3087472616e73666572876c766b5e527ac46c766b5ec3642c006c766b51c300c36c766b51c351c36c766b51c352c36165d40361527265c9016c766b57527ac4620e00006c766b57527ac46203006c766b57c3616c756653c56b6c766b00527ac4616168164e656f2e53746f726167652e476574436f6e746578746c766b00c3617c680f4e656f2e53746f726167652e4765746165700351936c766b51527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b51c361651103615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e7465787406737570706c79617c680f4e656f2e53746f726167652e4765746165f40251936c766b52527ac46168164e656f2e53746f726167652e476574436f6e7465787406737570706c796c766b52c361659302615272680f4e656f2e53746f726167652e50757461616c756653c56b6c766b00527ac461516c766b51527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b51c361654002615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e7465787406737570706c796c766b51c361650202615272680f4e656f2e53746f726167652e50757461516c766b52527ac46203006c766b52c3616c756659c56b6c766b00527ac46c766b51527ac46c766b52527ac4616168164e656f2e53746f726167652e476574436f6e746578746c766b00c3617c680f4e656f2e53746f726167652e4765746c766b53527ac46168164e656f2e53746f726167652e476574436f6e746578746c766b51c3617c680f4e656f2e53746f726167652e4765746c766b54527ac46c766b53c3616576016c766b52c3946c766b55527ac46c766b54c3616560016c766b52c3936c766b56527ac46c766b55c300a2640d006c766b52c300a2620400006c766b57527ac46c766b57c364ec00616168164e656f2e53746f726167652e476574436f6e746578746c766b00c36c766b55c36165d800615272680f4e656f2e53746f726167652e507574616168164e656f2e53746f726167652e476574436f6e746578746c766b51c36c766b56c361659c00615272680f4e656f2e53746f726167652e5075746155c57600135472616e73666572205375636365737366756cc476516c766b00c3c476526c766b51c3c476536c766b52c3c476546168184e656f2e426c6f636b636861696e2e476574486569676874c46168124e656f2e52756e74696d652e4e6f7469667961516c766b58527ac4620e00006c766b58527ac46203006c766b58c3616c756653c56b6c766b00527ac4616c766b00c36c766b51527ac46c766b51c36c766b52527ac46203006c766b52c3616c756653c56b6c766b00527ac461516c766b00c36a527a527ac46c766b51c36c766b52527ac46203006c766b52c3616c7566")
				if err != nil {
					panic(err)
				}
				return &result.ContractState{
					Version:     0,
					ScriptHash:  hash,
					Script:      script,
					ParamList:   []smartcontract.ParamType{smartcontract.ByteArrayType},
					ReturnType:  smartcontract.ByteArrayType,
					Name:        "Woolong",
					CodeVersion: "0.9.2",
					Author:      "lllwvlvwlll",
					Email:       "lllwvlvwlll@gmail.com",
					Description: "GO NEO!!!",
					Properties: result.Properties{
						HasStorage:       true,
						HasDynamicInvoke: false,
						IsPayable:        false,
					},
				}
			},
		},
	},
	"getnep5balances": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint160DecodeStringLE("1aada0032aba1ef6d1f07bbd8bec1d85f5380fb3")
				if err != nil {
					panic(err)
				}
				return c.GetNEP5Balances(hash)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"balance":[{"asset_hash":"a48b6e1291ba24211ad11bb90ae2a10bf1fcd5a8","amount":"50000000000","last_updated_block":251604}],"address":"AY6eqWjsUFCzsVELG7yG72XDukKvC34p2w"}}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint160DecodeStringLE("a48b6e1291ba24211ad11bb90ae2a10bf1fcd5a8")
				if err != nil {
					panic(err)
				}
				return &result.NEP5Balances{
					Balances: []result.NEP5Balance{{
						Asset:       hash,
						Amount:      "50000000000",
						LastUpdated: 251604,
					}},
					Address: "AY6eqWjsUFCzsVELG7yG72XDukKvC34p2w",
				}
			},
		},
	},
	"getnep5transfers": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetNEP5Transfers("AbHgdBaWEnHkCiLtDZXjhvhaAK2cwFh5pF")
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"sent":[],"received":[{"timestamp":1555651816,"asset_hash":"600c4f5200db36177e3e8a09e9f18e2fc7d12a0f","transfer_address":"AYwgBNMepiv5ocGcyNT4mA8zPLTQ8pDBis","amount":"1000000","block_index":436036,"transfer_notify_index":0,"tx_hash":"df7683ece554ecfb85cf41492c5f143215dd43ef9ec61181a28f922da06aba58"}],"address":"AbHgdBaWEnHkCiLtDZXjhvhaAK2cwFh5pF"}}`,
			result: func(c *Client) interface{} {
				assetHash, err := util.Uint160DecodeStringLE("600c4f5200db36177e3e8a09e9f18e2fc7d12a0f")
				if err != nil {
					panic(err)
				}
				txHash, err := util.Uint256DecodeStringLE("df7683ece554ecfb85cf41492c5f143215dd43ef9ec61181a28f922da06aba58")
				if err != nil {
					panic(err)
				}
				return &result.NEP5Transfers{
					Sent: []result.NEP5Transfer{},
					Received: []result.NEP5Transfer{
						{
							Timestamp:   1555651816,
							Asset:       assetHash,
							Address:     "AYwgBNMepiv5ocGcyNT4mA8zPLTQ8pDBis",
							Amount:      "1000000",
							Index:       436036,
							NotifyIndex: 0,
							TxHash:      txHash,
						},
					},
					Address: "AbHgdBaWEnHkCiLtDZXjhvhaAK2cwFh5pF",
				}
			},
		},
	},
	"getpeers": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetPeers()
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"unconnected":[{"address":"172.200.0.1","port":"20333"}],"connected":[{"address":"127.0.0.1","port":"20335"}],"bad":[{"address":"172.200.0.254","port":"20332"}]}}`,
			result: func(c *Client) interface{} {
				return &result.GetPeers{
					Unconnected: result.Peers{
						{
							Address: "172.200.0.1",
							Port:    "20333",
						},
					},
					Connected: result.Peers{
						{
							Address: "127.0.0.1",
							Port:    "20335",
						},
					},
					Bad: result.Peers{
						{
							Address: "172.200.0.254",
							Port:    "20332",
						},
					},
				}
			},
		},
	},
	"getrawmempool": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawMemPool()
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":["0x9786cce0dddb524c40ddbdd5e31a41ed1f6b5c8a683c122f627ca4a007a7cf4e"]}`,
			result: func(c *Client) interface{} {
				hash, err := util.Uint256DecodeStringLE("9786cce0dddb524c40ddbdd5e31a41ed1f6b5c8a683c122f627ca4a007a7cf4e")
				if err != nil {
					panic(err)
				}
				return []util.Uint256{hash}
			},
		},
	},
	"getrawtransaction": {
		{
			name: "positive",
			invoke: func(c *Client) (i interface{}, err error) {
				hash, err := util.Uint256DecodeStringLE("675b5bd2a90a1f5e74b2e4386162240318f86534f4d3061722ba78b4fe10fe53")
				if err != nil {
					panic(err)
				}
				return c.GetRawTransaction(hash)
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":"0000d5040000d60ac443bb800fb08261e75fa5925d747d4858610500000000000001fd040140947358ca2dd7543c3ff3f6ea1389a72c3d5ee99f47a9d0ef70bd84a9f57384e76271efc682f6741568c55907b1794b9f520f7d35f39382303bf0206945b5009a409f467419a886aebe6b482e6d5787981d98b58b82959a2858045bf5683665a5c25c502481b2d9655c902c5dcc147546bed58175c2ed16f328cc21e999e19741554063cab34f1613932947a1c346416b12b1ca724198016acc5fd760597539eed74f2069cfe2a8383e99595aefa3234d79d64a39e3f4c64e8cea800469a6f790999c408e2438fab244bdb79e67f6dab9cde0063e523bd0c175657a66e84897cd15eec8bf358661666679bf50334664872616faa366825f36873b16dd2add64c418cd5794534c2102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e4c2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd624c2102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc24c2103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee6995450683073b3bb"}`,
			result:         func(c *Client) interface{} { return &transaction.Transaction{} },
			check: func(t *testing.T, c *Client, result interface{}) {
				res, ok := result.(*transaction.Transaction)
				require.True(t, ok)
				assert.Equal(t, uint8(0), res.Version)
				assert.Equal(t, "675b5bd2a90a1f5e74b2e4386162240318f86534f4d3061722ba78b4fe10fe53", res.Hash().StringBE())
				assert.Equal(t, transaction.MinerType, res.Type)
				assert.Equal(t, false, res.Trimmed)
			},
		},
		{
			name: "verbose_positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("265f271088384b2f696e34bea0c8e02cf226351800c0866c1586be521536e579")
				if err != nil {
					panic(err)
				}
				return c.GetRawTransactionVerbose(hash)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"sys_fee":"0","net_fee":"0","blockhash":"0x66d1c140fbdc0eaa47e69a6a9c5034ebc3a449db98da565191ab863d1a079906","confirmations":205,"blocktime":1587379353,"txid":"0x79e5361552be86156c86c000183526f22ce0c8a0be346e692f4b388810275f26","size":437,"type":"MinerTransaction","version":0,"nonce":1237,"sender":"AZ81H31DMWzbSnFDLFkzh9vHwaDLayV7fU","valid_until_block":5,"attributes":[],"vin":[],"vout":[],"scripts":[{"invocation":"40f50121bb6ec9d8e0d1c15eea66b2ff7b51bb1bc4b3da27d9eac1d46b59e6a319bb1db4eb710c7f1931b0c2deaa2389a0fc3fe8c761cec40906b7973450c43173402dc082417a6815e722216de0b857eda6c846bf435088d543d2ab89f1dd92488e87b4d2c6508b0db945cbe6968e85c1c6d57274bfc898e82876c5cb08613da5d64053100f0162a41709a37305c300e7d6ac0d46575aab98dade7375b8d9ca980086594f1288dc68da0e0e42913d1c68024f63442a79c9478971d3ad93c5467ec53040a1c3a772a88b09cba8cc8ec3b46c0c0db6ac86519a7fd7db29b43d34e804a22d8839eaeb35e2a1e05d591fbad4ae290b90c6dc02dddbe28b2b3bf0fec2a337dd","verification":"532102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd622102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc22103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee69954ae"}]}}`,
			result: func(c *Client) interface{} {
				blockHash, err := util.Uint256DecodeStringLE("66d1c140fbdc0eaa47e69a6a9c5034ebc3a449db98da565191ab863d1a079906")
				if err != nil {
					panic(err)
				}
				sender, err := address.StringToUint160("AZ81H31DMWzbSnFDLFkzh9vHwaDLayV7fU")
				if err != nil {
					panic(err)
				}
				invocation, err := hex.DecodeString("40f50121bb6ec9d8e0d1c15eea66b2ff7b51bb1bc4b3da27d9eac1d46b59e6a319bb1db4eb710c7f1931b0c2deaa2389a0fc3fe8c761cec40906b7973450c43173402dc082417a6815e722216de0b857eda6c846bf435088d543d2ab89f1dd92488e87b4d2c6508b0db945cbe6968e85c1c6d57274bfc898e82876c5cb08613da5d64053100f0162a41709a37305c300e7d6ac0d46575aab98dade7375b8d9ca980086594f1288dc68da0e0e42913d1c68024f63442a79c9478971d3ad93c5467ec53040a1c3a772a88b09cba8cc8ec3b46c0c0db6ac86519a7fd7db29b43d34e804a22d8839eaeb35e2a1e05d591fbad4ae290b90c6dc02dddbe28b2b3bf0fec2a337dd")
				if err != nil {
					panic(err)
				}
				verification, err := hex.DecodeString("532102103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e2102a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd622102b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc22103d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee69954ae")
				if err != nil {
					panic(err)
				}
				tx := transaction.NewMinerTXWithNonce(1237)
				tx.ValidUntilBlock = 5
				tx.Sender = sender
				tx.Scripts = []transaction.Witness{
					{
						InvocationScript:   invocation,
						VerificationScript: verification,
					},
				}
				// Update hashes for correct result comparison.
				_ = tx.Hash()

				return &result.TransactionOutputRaw{
					Transaction: tx,
					TransactionMetadata: result.TransactionMetadata{
						SysFee:        0,
						NetFee:        0,
						Blockhash:     blockHash,
						Confirmations: 205,
						Timestamp:     uint64(1587379353),
					},
				}
			},
		},
	},
	"getstorage": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint160DecodeStringLE("03febccf81ac85e3d795bc5cbd4e84e907812aa3")
				if err != nil {
					panic(err)
				}
				key, err := hex.DecodeString("5065746572")
				if err != nil {
					panic(err)
				}
				return c.GetStorage(hash, key)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":"4c696e"}`,
			result: func(c *Client) interface{} {
				value, err := hex.DecodeString("4c696e")
				if err != nil {
					panic(err)
				}
				return value
			},
		},
	},
	"gettransactionheight": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("cb6ddb5f99d6af4c94a6c396d5294472f2eebc91a2c933e0f527422296fa9fb2")
				if err != nil {
					panic(err)
				}
				return c.GetTransactionHeight(hash)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":1}`,
			result: func(c *Client) interface{} {
				return uint32(1)
			},
		},
	},
	"gettxout": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("f4250dab094c38d8265acc15c366dc508d2e14bf5699e12d9df26577ed74d657")
				if err != nil {
					panic(err)
				}
				return c.GetTxOut(hash, 0)
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"N":0,"Asset":"c56f33fc6ecfcd0c225c4ab356fee59390af8560be0e930faebe74a6daff7c9b","Value":"2950","Address":"AHCNSDkh2Xs66SzmyKGdoDKY752uyeXDrt"}}`,
			result: func(c *Client) interface{} {
				return &result.TransactionOutput{
					N:       0,
					Asset:   "c56f33fc6ecfcd0c225c4ab356fee59390af8560be0e930faebe74a6daff7c9b",
					Value:   util.Fixed8FromInt64(2950),
					Address: "AHCNSDkh2Xs66SzmyKGdoDKY752uyeXDrt",
				}
			},
		},
	},
	"getunclaimed": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnclaimed("AGofsxAUDwt52KjaB664GYsqVAkULYvKNt")
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"available":750.032,"unavailable":2815.408,"unclaimed":3565.44}}`,
			result: func(c *Client) interface{} {
				return &result.Unclaimed{
					Available:   util.Fixed8FromFloat(750.032),
					Unavailable: util.Fixed8FromFloat(2815.408),
					Unclaimed:   util.Fixed8FromFloat(3565.44),
				}
			},
		},
	},
	"getunspents": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnspents("AK2nJJpJr6o664CWJKi1QRXjqeic2zRp8y")
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"balance":[{"unspent":[{"txid":"0x83df8bd085fcb60b2789f7d0a9f876e5f3908567f7877fcba835e899b9dea0b5","n":0,"value":"100000000"}],"asset_hash":"0xc56f33fc6ecfcd0c225c4ab356fee59390af8560be0e930faebe74a6daff7c9b","asset":"NEO","asset_symbol":"NEO","amount":"100000000"},{"unspent":[{"txid":"0x2ab085fa700dd0df4b73a94dc17a092ac3a85cbd965575ea1585d1668553b2f9","n":0,"value":"19351.99993"}],"asset_hash":"0x602c79718b16e442de58778e148d0b1084e3b2dffd5de6b7b16cee7969282de7","asset":"GAS","asset_symbol":"GAS","amount":"19351.99993"}],"address":"AK2nJJpJr6o664CWJKi1QRXjqeic2zRp8y"}}`,
			result:         func(c *Client) interface{} { return &result.Unspents{} },
			check: func(t *testing.T, c *Client, uns interface{}) {
				res, ok := uns.(*result.Unspents)
				require.True(t, ok)
				assert.Equal(t, "AK2nJJpJr6o664CWJKi1QRXjqeic2zRp8y", res.Address)
				assert.Equal(t, 2, len(res.Balance))
			},
		},
	},
	"getvalidators": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetValidators()
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":[{"publickey":"02b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc2","votes":"0","active":true},{"publickey":"02103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e","votes":"0","active":true},{"publickey":"03d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699","votes":"0","active":true},{"publickey":"02a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd62","votes":"0","active":true}]}`,
			result:         func(c *Client) interface{} { return []result.Validator{} },
			check: func(t *testing.T, c *Client, uns interface{}) {
				res, ok := uns.([]result.Validator)
				require.True(t, ok)
				assert.Equal(t, 4, len(res))
			},
		},
	},
	"getversion": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetVersion()
			},
			serverResponse: `{"id":1,"jsonrpc":"2.0","result":{"port":20332,"nonce":2153672787,"useragent":"/NEO-GO:0.73.1-pre-273-ge381358/"}}`,
			result: func(c *Client) interface{} {
				return &result.Version{
					Port:      uint16(20332),
					Nonce:     2153672787,
					UserAgent: "/NEO-GO:0.73.1-pre-273-ge381358/",
				}
			},
		},
	},
	"invokefunction": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint160DecodeStringLE("91b83e96f2a7c4fdf0c1688441ec61986c7cae26")
				if err != nil {
					panic(err)
				}
				return c.InvokeFunction("af7c7328eee5a275a3bcaee2bf0cf662b5e739be", "balanceOf", []smartcontract.Parameter{
					{
						Type:  smartcontract.Hash160Type,
						Value: hash,
					},
				})
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"script":"1426ae7c6c9861ec418468c1f0fdc4a7f2963eb89151c10962616c616e63654f6667be39e7b562f60cbfe2aebca375a2e5ee28737caf","state":"HALT","gas_consumed":"0.311","stack":[{"type":"ByteArray","value":"262bec084432"}],"tx":"d101361426ae7c6c9861ec418468c1f0fdc4a7f2963eb89151c10962616c616e63654f6667be39e7b562f60cbfe2aebca375a2e5ee28737caf000000000000000000000000"}}`,
			result: func(c *Client) interface{} {
				bytes, err := hex.DecodeString("262bec084432")
				if err != nil {
					panic(err)
				}
				return &result.Invoke{
					State:       "HALT",
					GasConsumed: "0.311",
					Script:      "1426ae7c6c9861ec418468c1f0fdc4a7f2963eb89151c10962616c616e63654f6667be39e7b562f60cbfe2aebca375a2e5ee28737caf",
					Stack: []smartcontract.Parameter{
						{
							Type:  smartcontract.ByteArrayType,
							Value: bytes,
						},
					},
				}
			},
		},
	},
	"invokescript": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return c.InvokeScript("00046e616d656724058e5e1b6008847cd662728549088a9ee82191")
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"script":"00046e616d656724058e5e1b6008847cd662728549088a9ee82191","state":"HALT","gas_consumed":"0.161","stack":[{"type":"ByteArray","value":"4e45503520474153"}],"tx":"d1011b00046e616d656724058e5e1b6008847cd662728549088a9ee82191000000000000000000000000"}}`,
			result: func(c *Client) interface{} {
				bytes, err := hex.DecodeString("4e45503520474153")
				if err != nil {
					panic(err)
				}
				return &result.Invoke{
					State:       "HALT",
					GasConsumed: "0.161",
					Script:      "00046e616d656724058e5e1b6008847cd662728549088a9ee82191",
					Stack: []smartcontract.Parameter{
						{
							Type:  smartcontract.ByteArrayType,
							Value: bytes,
						},
					},
				}
			},
		},
	},
	"sendrawtransaction": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SendRawTransaction(transaction.NewMinerTX())
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":true}`,
			result: func(c *Client) interface{} {
				// no error expected
				return nil
			},
		},
	},
	"submitblock": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SubmitBlock(block.Block{
					Base:         block.Base{},
					Transactions: nil,
					Trimmed:      false,
				})
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":true}`,
			result: func(c *Client) interface{} {
				// no error expected
				return nil
			},
		},
	},
	"validateaddress": {
		{
			name: "positive",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.ValidateAddress("AQVh2pG732YvtNaxEGkQUei3YA4cvo7d2i")
			},
			serverResponse: `{"jsonrpc":"2.0","id":1,"result":{"address":"AQVh2pG732YvtNaxEGkQUei3YA4cvo7d2i","isvalid":true}}`,
			result: func(c *Client) interface{} {
				// no error expected
				return nil
			},
		},
	},
}

type rpcClientErrorCase struct {
	name   string
	invoke func(c *Client) (interface{}, error)
}

var rpcClientErrorCases = map[string][]rpcClientErrorCase{
	`{"jsonrpc":"2.0","id":1,"result":"not-a-hex-string"}`: {
		{
			name: "getblock_not_a_hex_response",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndex(1)
			},
		},
		{
			name: "getblockheader_not_a_hex_response",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				return c.GetBlockHeader(hash)
			},
		},
		{
			name: "getrawtransaction_not_a_hex_response",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				return c.GetRawTransaction(hash)
			},
		},
		{
			name: "getstorage_not_a_hex_response",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint160DecodeStringLE("03febccf81ac85e3d795bc5cbd4e84e907812aa3")
				if err != nil {
					panic(err)
				}
				key, err := hex.DecodeString("5065746572")
				if err != nil {
					panic(err)
				}
				return c.GetStorage(hash, key)
			},
		},
	},
	`{"jsonrpc":"2.0","id":1,"result":"01"}`: {
		{
			name: "getblock_decodebin_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndex(1)
			},
		},
		{
			name: "getheader_decodebin_err",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				return c.GetBlockHeader(hash)
			},
		},
		{
			name: "getrawtransaction_decodebin_err",
			invoke: func(c *Client) (interface{}, error) {
				hash, err := util.Uint256DecodeStringLE("e93d17a52967f9e69314385482bf86f85260e811b46bf4d4b261a7f4135a623c")
				if err != nil {
					panic(err)
				}
				return c.GetRawTransaction(hash)
			},
		},
	},
	`{"jsonrpc":"2.0","id":1,"result":false}`: {
		{
			name: "sendrawtransaction_bad_server_answer",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SendRawTransaction(transaction.NewMinerTX())
			},
		},
		{
			name: "submitblock_bad_server_answer",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SubmitBlock(block.Block{
					Base:         block.Base{},
					Transactions: nil,
					Trimmed:      false,
				})
			},
		},
		{
			name: "validateaddress_bad_server_answer",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.ValidateAddress("AQVh2pG732YvtNaxEGkQUei3YA4cvo7d2i")
			},
		},
	},
	`{"id":1,"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid Params"}}`: {
		{
			name: "getaccountstate_invalid_params_error",
			invoke: func(c *Client) (i interface{}, err error) {
				return c.GetAccountState("")
			},
		},
		{
			name: "getapplicationlog_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetApplicationLog(util.Uint256{})
			},
		},
		{
			name: "getassetstate_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetAssetState(core.GoverningTokenID())
			},
		},
		{
			name: "getbestblockhash_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBestBlockHash()
			},
		},
		{
			name: "getblock_byindex_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndex(1)
			},
		},
		{
			name: "getblock_byindex_verbose_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndexVerbose(1)
			},
		},
		{
			name: "getblock_byhash_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByHash(util.Uint256{})
			},
		},
		{
			name: "getblock_byhash_verbose_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByHashVerbose(util.Uint256{})
			},
		},
		{
			name: "getblockhash_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHash(0)
			},
		},
		{
			name: "getblockheader_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHeader(util.Uint256{})
			},
		},
		{
			name: "getblockheader_verbose_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHeaderVerbose(util.Uint256{})
			},
		},
		{
			name: "getblocksysfee_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockSysFee(1)
			},
		},
		{
			name: "getclaimable_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetClaimable("")
			},
		},
		{
			name: "getconnectioncount_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetConnectionCount()
			},
		},
		{
			name: "getcontractstate_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetContractState(util.Uint160{})
			},
		},
		{
			name: "getnep5balances_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetNEP5Balances(util.Uint160{})
			},
		},
		{
			name: "getnep5transfers_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetNEP5Transfers("")
			},
		},
		{
			name: "getrawtransaction_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawTransaction(util.Uint256{})
			},
		},
		{
			name: "getrawtransaction_verbose_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawTransactionVerbose(util.Uint256{})
			},
		},
		{
			name: "getstorage_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetStorage(util.Uint160{}, []byte{})
			},
		},
		{
			name: "gettransactionheight_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetTransactionHeight(util.Uint256{})
			},
		},
		{
			name: "gettxoutput_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetTxOut(util.Uint256{}, 0)
			},
		},
		{
			name: "getunclaimed_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnclaimed("")
			},
		},
		{
			name: "getunspents_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnspents("")
			},
		},
		{
			name: "invokefunction_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.InvokeFunction("", "", []smartcontract.Parameter{})
			},
		},
		{
			name: "invokescript_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.InvokeScript("")
			},
		},
		{
			name: "sendrawtransaction_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SendRawTransaction(&transaction.Transaction{})
			},
		},
		{
			name: "submitblock_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SubmitBlock(block.Block{})
			},
		},
		{
			name: "validateaddress_invalid_params_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.ValidateAddress("")
			},
		},
	},
	`{}`: {
		{
			name: "getaccountstate_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetAccountState("")
			},
		},
		{
			name: "getapplicationlog_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetApplicationLog(util.Uint256{})
			},
		},
		{
			name: "getassetstate_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetAssetState(core.GoverningTokenID())
			},
		},
		{
			name: "getbestblockhash_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBestBlockHash()
			},
		},
		{
			name: "getblock_byindex_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndex(1)
			},
		},
		{
			name: "getblock_byindex_verbose_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByIndexVerbose(1)
			},
		},
		{
			name: "getblock_byhash_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByHash(util.Uint256{})
			},
		},
		{
			name: "getblock_byhash_verbose_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockByHashVerbose(util.Uint256{})
			},
		},
		{
			name: "getblockcount_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockCount()
			},
		},
		{
			name: "getblockhash_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHash(1)
			},
		},
		{
			name: "getblockheader_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHeader(util.Uint256{})
			},
		},
		{
			name: "getblockheader_verbose_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockHeaderVerbose(util.Uint256{})
			},
		},
		{
			name: "getblocksysfee_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetBlockSysFee(1)
			},
		},
		{
			name: "getclaimable_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetClaimable("")
			},
		},
		{
			name: "getconnectioncount_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetConnectionCount()
			},
		},
		{
			name: "getcontractstate_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetContractState(util.Uint160{})
			},
		},
		{
			name: "getnep5balances_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetNEP5Balances(util.Uint160{})
			},
		},
		{
			name: "getnep5transfers_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetNEP5Transfers("")
			},
		},
		{
			name: "getpeers_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetPeers()
			},
		},
		{
			name: "getrawmempool_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawMemPool()
			},
		},
		{
			name: "getrawtransaction_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawTransaction(util.Uint256{})
			},
		},
		{
			name: "getrawtransaction_verbose_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetRawTransactionVerbose(util.Uint256{})
			},
		},
		{
			name: "getstorage_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetStorage(util.Uint160{}, []byte{})
			},
		},
		{
			name: "gettransactionheight_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetTransactionHeight(util.Uint256{})
			},
		},
		{
			name: "getxoutput_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetTxOut(util.Uint256{}, 0)
			},
		},
		{
			name: "getunclaimed_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnclaimed("")
			},
		},
		{
			name: "getunspents_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetUnspents("")
			},
		},
		{
			name: "getvalidators_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetValidators()
			},
		},
		{
			name: "getversion_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.GetVersion()
			},
		},
		{
			name: "invokefunction_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.InvokeFunction("", "", []smartcontract.Parameter{})
			},
		},
		{
			name: "invokescript_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return c.InvokeScript("")
			},
		},
		{
			name: "sendrawtransaction_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SendRawTransaction(transaction.NewMinerTX())
			},
		},
		{
			name: "submitblock_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.SubmitBlock(block.Block{
					Base:         block.Base{},
					Transactions: nil,
					Trimmed:      false,
				})
			},
		},
		{
			name: "validateaddress_unmarshalling_error",
			invoke: func(c *Client) (interface{}, error) {
				return nil, c.ValidateAddress("")
			},
		},
	},
}

func TestRPCClient(t *testing.T) {
	for method, testBatch := range rpcClientTestCases {
		t.Run(method, func(t *testing.T) {
			for _, testCase := range testBatch {
				t.Run(testCase.name, func(t *testing.T) {
					srv := initTestServer(t, testCase.serverResponse)
					defer srv.Close()

					endpoint := srv.URL
					opts := Options{}
					c, err := New(context.TODO(), endpoint, opts)
					if err != nil {
						t.Fatal(err)
					}

					actual, err := testCase.invoke(c)
					assert.NoError(t, err)

					expected := testCase.result(c)
					if testCase.check == nil {
						assert.Equal(t, expected, actual)
					} else {
						testCase.check(t, c, actual)
					}
				})
			}
		})
	}
	for serverResponse, testBatch := range rpcClientErrorCases {
		srv := initTestServer(t, serverResponse)
		defer srv.Close()

		endpoint := srv.URL
		opts := Options{}
		c, err := New(context.TODO(), endpoint, opts)
		if err != nil {
			t.Fatal(err)
		}

		for _, testCase := range testBatch {
			t.Run(testCase.name, func(t *testing.T) {
				_, err := testCase.invoke(c)
				assert.Error(t, err)
			})
		}
	}
}

func initTestServer(t *testing.T, resp string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestHandler(t, w, resp)
	}))

	return srv
}

func requestHandler(t *testing.T, w http.ResponseWriter, resp string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(json.RawMessage(resp))

	if err != nil {
		t.Fatalf("Error encountered while encoding response: %s", err.Error())
	}
}

func TestCalculateValidUntilBlock(t *testing.T) {
	var (
		getBlockCountCalled int
		getValidatorsCalled int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r := request.NewIn()
		err := r.DecodeData(req.Body)
		if err != nil {
			t.Fatalf("Cannot decode request body: %s", req.Body)
		}
		var response string
		switch r.Method {
		case "getblockcount":
			getBlockCountCalled++
			response = `{"jsonrpc":"2.0","id":1,"result":50}`
		case "getvalidators":
			getValidatorsCalled++
			response = `{"id":1,"jsonrpc":"2.0","result":[{"publickey":"02b3622bf4017bdfe317c58aed5f4c753f206b7db896046fa7d774bbc4bf7f8dc2","votes":"0","active":true},{"publickey":"02103a7f7dd016558597f7960d27c516a4394fd968b9e65155eb4b013e4040406e","votes":"0","active":true},{"publickey":"03d90c07df63e690ce77912e10ab51acc944b66860237b608c4f8f8309e71ee699","votes":"0","active":true},{"publickey":"02a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd62","votes":"0","active":true}]}`
		default:
			t.Fatalf("Bad request method: %s", r.Method)
		}
		requestHandler(t, w, response)
	}))
	defer srv.Close()

	endpoint := srv.URL
	opts := Options{}
	c, err := New(context.TODO(), endpoint, opts)
	if err != nil {
		t.Fatal(err)
	}

	validUntilBlock, err := c.CalculateValidUntilBlock()
	assert.NoError(t, err)
	assert.Equal(t, uint32(54), validUntilBlock)
	assert.Equal(t, 1, getBlockCountCalled)
	assert.Equal(t, 1, getValidatorsCalled)

	// check, whether caching is working
	validUntilBlock, err = c.CalculateValidUntilBlock()
	assert.NoError(t, err)
	assert.Equal(t, uint32(54), validUntilBlock)
	assert.Equal(t, 2, getBlockCountCalled)
	assert.Equal(t, 1, getValidatorsCalled)
}
