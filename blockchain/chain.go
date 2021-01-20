package blockchain

// Chain is actually a convenience for the concept of a chain.
// We keep a fixed number of blocks and speculate on blocks we recieve.
// The consensus mechanism says that the network may broadcast a new
// chain with a higher depth than ours. At this point we need to verify that the
// new chain is indeed valid and longer than ours.
type Chain struct {
	id   []byte       // the chainID
	head *BlockHeader // The current head block that we know about
}
