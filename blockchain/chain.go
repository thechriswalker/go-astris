package blockchain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/rs/zerolog/log"
)

// Chain is actually a convenience for the concept of a chain.
// We keep a fixed number of blocks and speculate on blocks we recieve.
// The consensus mechanism says that the network may broadcast a new
// chain with a higher depth than ours. At this point we need to verify that the
// new chain is indeed valid and longer than ours.
type Chain interface {
	ID() BlockID                 // get the chain ID of this chain (the ID of the genesis block)
	Head() (*BlockHeader, error) // get the current head of the chain

	// GetHeader retrieves a block from the current chain
	// with the given hash
	Header(hash BlockID) (*BlockHeader, error)
	// GetPayload retrieves the payload for a block with the given hash
	Payload(hash BlockID) ([]byte, error)

	// given a certain depth, find the ID of the block at that depth, or not.
	AtDepth(d uint64) (BlockID, bool)

	// give a blockID, find the "next" block, or not if there isn't one
	Next(prev BlockID) (BlockID, bool)

	// Add a block to the chain
	// the block should be fully populated and implementations should validate that
	// This is now an "official" block and so we add to the current state by "validating" it.
	Add(*Block) error
	// mint a new block with work level w and timestamp ts, then add it to the chain
	Mint(b *Block, w int, ts uint32) error
}

type blockchain struct {
	id        BlockID    // the election id
	head      *blocklist // the blocklist going from the current head to the genesis block
	depth     uint64
	db        Storage        // the backing storage
	validator BlockValidator // the block validator
}

func (bc *blockchain) ID() BlockID {
	return bc.id
}

func (bc *blockchain) Head() (*BlockHeader, error) {
	if bc.head == nil {
		return nil, nil // no head, but no error
	}
	return bc.Header(bc.head.blockId)
}

func (bc *blockchain) Header(id BlockID) (*BlockHeader, error) {
	return bc.db.Header(id, &BlockHeader{})
}

func (bc *blockchain) Payload(id BlockID) ([]byte, error) {
	return bc.db.Payload(id, nil)
}

func (bc *blockchain) Add(blk *Block) error {
	// we need to validate the block.
	// note, statefully validator should only update state for valid blocks.
	if blk.Header.Depth != bc.depth+1 {
		return fmt.Errorf("Add block failed, invalid depth %d expected %d", blk.Header.Depth, bc.depth+1)
	}
	// so this is safe to call.
	if err := bc.validator.Validate(blk); err != nil {
		return fmt.Errorf("Add block failed, block invalid: %w", err)
	}
	// add to persistent storage
	if err := bc.db.Write(blk); err != nil {
		return fmt.Errorf("Add block failed, storage errror: %w", err)
	}
	// add, this means update the blocklist
	newHead := &blocklist{
		prev:    bc.head, // current head becomes previous
		blockId: blk.Header.ID,
	}

	// link the current head's next to this node
	bc.head.next = newHead
	// and replace the chain head.
	bc.head = newHead
	bc.depth++

	return nil
}

// this is for adding a new block we just created, we need to link
// the IDs and do the PoW before we can add it to the chain.
func (bc *blockchain) Mint(blk *Block, workLevel int, ts uint32) error {
	blk.Header.PrevID = bc.head.blockId
	blk.Header.Depth = bc.depth + 1
	var found bool
	if ts == 0 {
		ts = uint32(time.Now().Unix())
	}
	ctx := context.TODO()
	for !found {
		blk.Header.EpochSeconds = ts
		blk.Header.Proof, found = blk.Header.CalculateProofOfWork(ctx, workLevel)
		ts++
	}
	// now add the id
	blk.Header.ID = blk.Header.CalculateBlockID()
	return bc.Add(blk)
}

// we look back through our linked list to find the block at the required depth
func (bc *blockchain) AtDepth(d uint64) (id BlockID, ok bool) {
	if d == 0 {
		// that is the genesis.
		return bc.id, true
	}
	curr := bc.head
	currDepth := bc.depth

	for {
		if currDepth < d {
			// we don't have this depth
			return id, false
		}
		if curr == nil {
			// we didn't find it, which is worrying
			panic(fmt.Sprintf("Could not find block at depth %d", d))
		}
		if currDepth == d {
			return curr.blockId, true
		}
		// depth is less
		curr = curr.prev
		currDepth--
	}
}

func (bc *blockchain) Next(id BlockID) (nextId BlockID, ok bool) {
	// go back until we find the block in question.
	curr := bc.head
	for {
		if curr == nil {
			// could not find the id.
			return nextId, false
		}
		if curr.blockId == id {
			// we found it, return the "nextId" or not if there isn't one
			if curr.next == nil {
				// no next, yet
				return nextId, false
			}
			return curr.next.blockId, true
		}
		curr = curr.prev
	}
}

// exposes a channel with an "on-close" notification
// to stop sending blocks on the channel.
// it is a fanout pub-sub pattern.

type Storage interface {
	Head() (BlockID, error)                                   // find the current HEAD
	Header(id BlockID, hd *BlockHeader) (*BlockHeader, error) // get Header for block into hd
	Payload(id BlockID, b []byte) ([]byte, error)             // get Payload for block into b
	Write(*Block) error                                       // write new block
}

// BlockValidator is a "stateful" system, it should always be
// expecting the "next" block.
// But we should have a way to clone that state at a point in time (hopefully cheaply)
// so we can speculatively validate without commiting.
// That is what the Clone method is for
type BlockValidator interface {
	Validate(*Block) error
	WorkLevel() int
}

// this validator does nothing but validate that the payload matches the hash.
// all other validators should do that. as well
type PayloadHashValidator int

// stateful work level
func (p PayloadHashValidator) WorkLevel() int {
	return int(p)
}

// stateless hash validation
func (p PayloadHashValidator) Validate(b *Block) error {
	sum := sha256.Sum256(b.Payload)
	if sum != b.Header.PayloadHash {
		return fmt.Errorf("Payload Hash doesn't match hash(Payload): %x != %x", b.Header.PayloadHash, sum)
	}
	return nil
}

type chain struct {
	id   BlockID
	head BlockID
	db   Storage
}

type blocklist struct {
	prev    *blocklist
	next    *blocklist
	blockId BlockID
	depth   uint64
}

func getStorage(dir string, chainId BlockID) (*SQLiteStorage, error) {
	dbname := fmt.Sprintf("chain_%s.db", chainId)
	// we need to open the database for this chain.
	// if it doesn't exist, create it.
	// if it does exist, validate it.
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("Could not create parent directory for blockchain (%s): %w", dir, err)
	}
	p := filepath.Join(dir, dbname)
	return NewSQLiteStorage(p)
}

func Create(dir string, genesisBlock *Block, initialWorkLevel int, validator BlockValidator) (Chain, error) {
	chainId := genesisBlock.Header.ID
	db, err := getStorage(dir, chainId)
	if err != nil {
		return nil, err
	}
	// write the block we no checks.
	if err := db.Write(genesisBlock); err != nil {
		return nil, err
	}
	//	pass to the rest of the open function.
	return openChain(db, chainId, initialWorkLevel, validator)
}

func Open(dir string, chainId BlockID, initialWorkLevel int, validator BlockValidator) (Chain, error) {
	db, err := getStorage(dir, chainId)
	if err != nil {
		return nil, err
	}
	return openChain(db, chainId, initialWorkLevel, validator)
}

func openChain(db *SQLiteStorage, chainId BlockID, initialWorkLevel int, validator BlockValidator) (Chain, error) {
	// validate chain!
	head, err := db.Head()
	if err != nil {
		return nil, err
	}

	if validator == nil {
		validator = PayloadHashValidator(initialWorkLevel)
	}

	if head == ZeroId {
		log.Debug().Msgf("Loaded fresh chain for election: %s", chainId)

		return &blockchain{
			id:        chainId,
			db:        db,
			validator: validator,
			head:      nil, // no head yet
			depth:     0,
		}, nil
	}
	// start with the head and work backwards.
	// while we do that, we build the chain as a doubly linked list.
	// that way we can iterate one way, then back
	curr := &blocklist{blockId: head}
	log.Debug().Msgf("Validating chain starting at block %s", head)
	start := time.Now()
	header := &BlockHeader{}
	lastDepth := uint64(0)
	fullDepth := -1
	for {
		// lookup block header
		_, err = db.Header(curr.blockId, header)
		if err != nil {
			return nil, fmt.Errorf("Chain invalid [block=%s] error reading block header: %w", curr.blockId, err)
		}
		// if fullDepth is -1 then this is the first block.
		if fullDepth == -1 {
			fullDepth = int(header.Depth)
		}
		// we found the header, is the header valid?
		if lastDepth != 0 && header.Depth != lastDepth-1 {
			return nil, fmt.Errorf("Chain invalid [block=%s] expecting block depth %d, found %d", curr.blockId, lastDepth-1, header.Depth)
		}
		lastDepth = header.Depth

		// OK all looks good. Let's update our list
		// with the "next" block unless this was the genesis block.
		if header.IsGenesis() {
			// for this last block, the id should be the chain id.
			if header.ID != chainId {
				return nil, fmt.Errorf("Chain invalid [block=%s] genesis block ID does not match chainID (%s)", curr.blockId, chainId)
			}
			// all good!
			break
		}
		prev := &blocklist{
			next:    curr,
			blockId: header.PrevID,
		}
		curr.prev = prev
		// switch
		curr = prev
	}
	pass1 := time.Now()
	log.Info().Dur("pass1_ms", pass1.Sub(start)).Msg("Chain reverse pass complete, starting forward validation")
	bar := MaybeProgress(fullDepth)
	bar.Start()
	// if we get this far the chain of headers is valid and complete.
	// now we need to reverse back up the chain validating payloads
	blk := &Block{Header: &BlockHeader{}}

	for {
		// now go back UP the chain. First we validate this block.
		blk.Header, err = db.Header(curr.blockId, blk.Header)
		if err != nil {
			return nil, fmt.Errorf("Chain invalid [block=%s] error reading block header: %w", curr.blockId, err)
		}
		blk.Payload, err = db.Payload(curr.blockId, blk.Payload)
		if err != nil {
			return nil, fmt.Errorf("Chain invalid [block=%s] error reading block payload: %w", curr.blockId, err)
		}
		// we have to validate proof-of-work "forwards"
		if err = blk.Header.Validate(validator.WorkLevel()); err != nil {
			return nil, fmt.Errorf("Chain invalid [block=%s] header validation failed: %w", curr.blockId, err)
		}
		// we can now run the validator, which needs to run from depth=0 upwards.
		if err = validator.Validate(blk); err != nil {
			return nil, fmt.Errorf("Chain invalid [block=%s] error validating block: %w", curr.blockId, err)
		}
		// we need to get back to the head, which will have no "next"
		if curr.next == nil {
			break // All good and `curr` is at the head of this chain.
		}
		// move on one
		curr = curr.next
		bar.Increment()
	}
	pass2 := time.Now()
	totalms := pass2.Sub(start)
	bar.Finish()
	log.Info().
		Dur("total_ms", totalms).
		Dur("pass2_ms", pass2.Sub(pass1)).
		Msg("Chain forward pass complete")

	log.Info().
		Str("chain", chainId.String()).
		Str("head", curr.blockId.String()).
		Uint64("depth", blk.Header.Depth).
		Dur("ms", totalms).
		Msg("Blockchain Validation Success")

	// all good!
	return &blockchain{
		id:        chainId,
		head:      curr,
		db:        db,
		validator: validator,
		depth:     blk.Header.Depth,
	}, nil

}

type maybeProgress struct {
	bar *pb.ProgressBar
}

func MaybeProgress(n int) *maybeProgress {
	mp := &maybeProgress{}
	if n > 1000 {
		mp.bar = pb.ProgressBarTemplate(`{{string . "prefix"}}{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{etime . }`).New(n)
		mp.bar.SetRefreshRate(time.Second)
	}
	return mp
}

func (mp *maybeProgress) Start() {
	if mp.bar != nil {
		mp.bar.Start()
	}
}
func (mp *maybeProgress) Increment() {
	if mp.bar != nil {
		mp.bar.Increment()
	}
}

func (mp *maybeProgress) Finish() {
	if mp.bar != nil {
		mp.bar.Finish()
	}
}
