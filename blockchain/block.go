package blockchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"math"
)

// Block in the blockchain
type Block struct {
	Header  *BlockHeader
	Payload []byte
}

// BlockHeader is the detail of a block without the payload.
// We only ever keep the headers in memory. We read the payload
// on demand, or when validating/iterating the chain.
type BlockHeader struct {
	ID           []byte
	PrevID       []byte
	ChainID      []byte
	EpochSeconds uint64
	PayloadHash  []byte
	// these 2 fields are not computed in the hash
	Nonce uint32
	Depth int
}

// Validate a BlockHeader, assuming the PrevId and PayloadHash are good.
// This means you will need to validate the PayloadHash matches the payload
// seperately.
// It _will_ validate the proof of work though as that is not included in the hash
func (bh *BlockHeader) Validate(workLevel int) bool {
	if !bh.VerifyProofOfWork(workLevel) {
		return false
	}
	expected := bh.CalculateBlockID(nil)
	actual := bh.ID
	return bytes.Compare(expected, actual) == 0
}

// CalculateBlockID works out the hash given the current
// state of the header and puts it in the slice given.
// This can be `nil` and a new slice will be allocated and returned.
func (bh *BlockHeader) CalculateBlockID(b []byte) []byte {
	// to calculate the header ID we take the hash of
	// the previous block and the payload hash
	// then add the epoch seconds
	// NB the proof of work nonce is NOT part of the ID
	// as it is worked out seperately. This means that validating
	// a block MUST validate the proof of work AND the ID
	h := sha256.New()
	h.Write(bh.PrevID)
	h.Write(bh.PayloadHash)
	// now the seconds and nonce as BigEndian Uints.
	binary.Write(h, binary.BigEndian, bh.EpochSeconds)
	binary.Write(h, binary.BigEndian, bh.Nonce)
	return h.Sum(b)
}

// Proof of work functions. These are fixed for this proof of concept
// and use a simple HashCash-esque function.

// VerifyProofOfWork checks the current Nonce is correct
func (bh *BlockHeader) VerifyProofOfWork(n int) bool {
	return proofOfWorkCheck(n, sha256.New(), make([]byte, 0, sha256.Size), bh, bh.Nonce)
}

// CalculateProofOfWork tries to calculate a proof of work with the current block
// if the context if cancelled it will return early.
// if it cannot it will return false as the bool
func (bh *BlockHeader) CalculateProofOfWork(ctx context.Context, n int) (nonce uint32, found bool) {
	h := sha256.New()
	s := make([]byte, sha256.Size)
	cancel := ctx.Done()
	for {
		// check we wish to continue
		select {
		case <-cancel:
			// bail
			nonce = 0
			return
		default:
			// proceed one iteration
		}
		if proofOfWorkCheck(n, h, s, bh, nonce) {
			found = true
			return
		}
		// if we exhaust all possibilities we should return and the caller can decide what to do.
		// usually it means incrementing the timestamp on the block (and so the ID) and then trying
		// again.
		if nonce == math.MaxUint32 {
			nonce = 0
			return
		}
		nonce++
	}
}

// This is our proof of work check.
// we create a hash of the previous ID, the current ID and our nonce.
// that hash should have at least `workLevel` zeros at the start.
func proofOfWorkCheck(workLevel int, h hash.Hash, sum []byte, bh *BlockHeader, p uint32) bool {
	h.Reset()
	h.Write(bh.PrevID)
	h.Write(bh.ID)
	binary.Write(h, binary.BigEndian, p)
	sum = h.Sum(sum[0:0])
	// does the Sum have the leading zero bytes
	for i := 0; i < workLevel; i++ {
		if sum[i] != 0 {
			return false
		}
	}
	return true
}
