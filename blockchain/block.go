package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"math/bits"
	"strings"
	"time"
)

type BlockID [sha256.Size]byte

func (id BlockID) String() string {
	return base64.RawURLEncoding.EncodeToString(id[:])
}
func (id *BlockID) FromString(s string) error {
	s = strings.TrimRight(s, "=")
	l := base64.RawURLEncoding.DecodedLen(len(s))

	if l != len(id) {
		return fmt.Errorf("ID is not the correct length for a base64url encoded BlockID")
	}
	n, err := base64.RawURLEncoding.Decode(id[:], []byte(s))
	if err != nil {
		return fmt.Errorf("ID is not valid base64url encoded: %w", err)
	}
	if n != len(id) {
		return fmt.Errorf("ID is not valid base64url encoded: incorrect length")
	}
	return nil
}

// Block in the blockchain
type Block struct {
	Header  *BlockHeader
	Payload []byte
}

// BlockHeader is the detail of a block without the payload.
// We only ever keep the headers in memory. We read the payload
// on demand, or when validating/iterating the chain.
type BlockHeader struct {
	ID           BlockID
	PrevID       BlockID
	EpochSeconds uint32
	PayloadHash  [sha256.Size]byte
	PayloadHint  uint8
	Proof        uint32
	Depth        uint64
}

// Go zero's buffers on allocation
var ZeroId BlockID

// IsGenesis checks if this is the initial block
func (bh *BlockHeader) IsGenesis() bool {
	return bh.Depth == 0 && bh.PrevID == ZeroId
}

// Validate a BlockHeader, assuming the PrevId and PayloadHash are good.
// This means you will need to validate the PayloadHash matches the payload
// seperately.
// It _will_ validate the proof of work though as that is not included in the hash
func (bh *BlockHeader) Validate(workLevel int) error {
	expected := bh.CalculateBlockID()
	if expected == bh.ID {
		// OK the id checks out, does it pass the proof of work check?
		if bh.VerifyProofOfWork(workLevel) {
			return nil
		}
		return fmt.Errorf("Proof of work invalid (level=%d, hexid=%x", workLevel, bh.ID[:])
	}
	return fmt.Errorf("BlockID invalid for header")
}

func (bh *BlockHeader) GetTime() time.Time {
	return time.Unix(int64(bh.EpochSeconds), 0)
}

// CalculateBlockID works out the hash given the current
// state of the header and puts it in the slice given.
// This can be `nil` and a new slice will be allocated and returned.
func (bh *BlockHeader) CalculateBlockID() (id BlockID) {
	// work out the header with the _actual_ proof we have
	calculateBlockHash(sha256.New(), &id, bh, bh.Proof)
	return id
}

// Proof of work functions. These are fixed for this proof of concept
// and use a simple HashCash-esque function.
// Note that we want to be able to verify a block at a time without any other data.
// So the Nonce must be part of the block ID, or we could create multiple blocks simultaneously.
// The only other data from the previous block is the previous ID, so we calculate the

// VerifyProofOfWork checks the current Nonce is correct
func (bh *BlockHeader) VerifyProofOfWork(n int) bool {
	return proofOfWorkCheck(n, &(bh.ID))
}

// CalculateProofOfWork tries to calculate a proof of work with the current block
// if the context if cancelled it will return early.
// if it cannot it will return false as the bool
func (bh *BlockHeader) CalculateProofOfWork(ctx context.Context, n int) (nonce uint32, found bool) {
	h := sha256.New()
	var s BlockID
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
		calculateBlockHash(h, &s, bh, nonce)
		if proofOfWorkCheck(n, &s) {
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
// we check that the hash we are given is "less-than" the work level.
// that hash should have at least `workLevel` zero bits at the start.
// sum must be non-nil and at least `ceiling(workLevel/8)`` bytes long
// we don't check that though.
func proofOfWorkCheck(workLevel int, id *BlockID) bool {
	// we split this into 64bit ints, any full 64s will just
	// check against 0
	i := 0
	// read the first one and check worklevel
	for workLevel > 64 {
		// this should be all zeros, so `0`
		if binary.BigEndian.Uint64(id[i:i+8]) != 0 {
			return false
		}
		// more on 8 bytes
		i += 8
		// and reduce the workLevel 64 bits
		workLevel -= 64
		// which shifts our view one uint64 into the `sum` and
		// lowers the workLevel to account for the bits we have checked
	}
	// Go has an optimised function for counting the leading zeros
	// of a 64 bit unsigned int
	x := binary.BigEndian.Uint64(id[i : i+8])
	return bits.LeadingZeros64(x) >= workLevel
}

func calculateBlockHash(h hash.Hash, sum *BlockID, bh *BlockHeader, p uint32) {
	h.Reset()
	h.Write(bh.PrevID[:])
	h.Write(bh.PayloadHash[:])
	// now the seconds and nonce as BigEndian Uints.
	binary.Write(h, binary.BigEndian, bh.EpochSeconds)
	binary.Write(h, binary.BigEndian, bh.PayloadHint)
	binary.Write(h, binary.BigEndian, p)
	h.Sum(sum[0:0])
}
