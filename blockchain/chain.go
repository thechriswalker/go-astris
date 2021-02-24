package blockchain

// Chain is actually a convenience for the concept of a chain.
// We keep a fixed number of blocks and speculate on blocks we recieve.
// The consensus mechanism says that the network may broadcast a new
// chain with a higher depth than ours. At this point we need to verify that the
// new chain is indeed valid and longer than ours.
type Chain interface {
	ID() []byte         // get the chain ID of this chain
	Head() *BlockHeader // get the current head of the chain

	// GetHeader retrieves a block from the current chain
	// with the given hash
	GetHeader(hash []byte) (*BlockHeader, error)
	// GetPayload retrieves the payload for a block with the given hash
	GetPayload(hash []byte) ([]byte, error)

	// Extend the chain with the given blocks.
	// NB this will fail unless:
	//   1. the blocks self-validate from the end to the beginning
	//   2. the first block references a valid prev hash that exists
	//   3. the first block depth is the prev+1
	//   4. the final block depth is greater than the current HEAD
	//
	// That is, this subchain correctly extends the current chain and is longer.
	Extend(blocks ...*Block) error

	// OnHeadUpdate allows you to react to updates to the Head of the chain.
	// pass in the function to react (it should not block) and the return
	// value is the unsubscribe function.
	OnHeadUpdate(func(newHead *BlockHeader)) (unsubscribe func())
}

// exposes a channel with an "on-close" notification
// to stop sending blocks on the channel.
// it is a fanout pub-sub pattern.
