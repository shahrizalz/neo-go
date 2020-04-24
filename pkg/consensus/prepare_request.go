package consensus

import (
	"github.com/nspcc-dev/dbft/payload"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

// prepareRequest represents dBFT prepareRequest message.
type prepareRequest struct {
	timestamp         uint64
	nonce             uint64
	transactionHashes []util.Uint256
	nextConsensus     util.Uint160
}

var _ payload.PrepareRequest = (*prepareRequest)(nil)

// EncodeBinary implements io.Serializable interface.
func (p *prepareRequest) EncodeBinary(w *io.BinWriter) {
	w.WriteU64LE(p.timestamp)
	w.WriteU64LE(p.nonce)
	w.WriteBytes(p.nextConsensus[:])
	w.WriteArray(p.transactionHashes)
}

// DecodeBinary implements io.Serializable interface.
func (p *prepareRequest) DecodeBinary(r *io.BinReader) {
	p.timestamp = r.ReadU64LE()
	p.nonce = r.ReadU64LE()
	r.ReadBytes(p.nextConsensus[:])
	r.ReadArray(&p.transactionHashes)
}

// Timestamp implements payload.PrepareRequest interface.
func (p *prepareRequest) Timestamp() uint64 { return p.timestamp * 1000000 }

// SetTimestamp implements payload.PrepareRequest interface.
func (p *prepareRequest) SetTimestamp(ts uint64) { p.timestamp = ts / 1000000 }

// Nonce implements payload.PrepareRequest interface.
func (p *prepareRequest) Nonce() uint64 { return p.nonce }

// SetNonce implements payload.PrepareRequest interface.
func (p *prepareRequest) SetNonce(nonce uint64) { p.nonce = nonce }

// TransactionHashes implements payload.PrepareRequest interface.
func (p *prepareRequest) TransactionHashes() []util.Uint256 { return p.transactionHashes }

// SetTransactionHashes implements payload.PrepareRequest interface.
func (p *prepareRequest) SetTransactionHashes(hs []util.Uint256) { p.transactionHashes = hs }

// NextConsensus implements payload.PrepareRequest interface.
func (p *prepareRequest) NextConsensus() util.Uint160 { return p.nextConsensus }

// SetNextConsensus implements payload.PrepareRequest interface.
func (p *prepareRequest) SetNextConsensus(nc util.Uint160) { p.nextConsensus = nc }
