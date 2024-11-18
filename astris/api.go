package astris

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"github.com/thechriswalker/go-astris/blockchain"
)

type ID = blockchain.BlockID

const IDSize = sha256.Size

type P2PNode interface {
	// these are handlers for the gRPC _server_ interface

	// Peer functions
	PeerSeen(net.Addr)                                         // update the time of last seen this peer
	ShouldBlacklist(net.Addr) bool                             // check whether this peer is bad
	BadPeer(net.Addr)                                          // mark this peer as bad, dishonest
	GetPeers(context.Context, func(peer net.Addr) error) error // react to peer events with a callback.

	// block functions.
	GetBlockWithPayload(ID) (*blockchain.Block, error)                             // fetch a specific block
	GetBlockHeaderAtDepth(uint64) (*blockchain.BlockHeader, error)                 // find the header of the block at a given depth
	StreamBlocksFromID(context.Context, ID, func(b *blockchain.Block) error) error // get all blocks in order _after_ the given one
	// this is for recvBlocks so we can send them, it sends the current "head" immediately
	StreamNewBlocks(context.Context, func(*blockchain.BlockHeader) error) error

	// this is for publish block, but can be used internally (like for voting)
	NewBlock(*blockchain.Block) (bool, error) // check new block, bool=true if block accepted, error means internal problem
}

// @TODO This needs speculative addition to the chain
// i.e. the ability to accept multiple chains of different depths
// How can we do this, it is super complicated.
// Our validator needs to be in the context of a specific
// chain-branch. Perhaps our blockchain.Chain interface implementation
// have have an "in-memory fork", and then we can have a "commit"
// function to add the speculative blocks later.
//
// We will need to keep checkpoints of current validation status (external to the chain)
// where we will reference block-depth and block-id and validation status (i.e. cumulative state)
// when we load a chain database, we will validate against our checkpoint DB to ensure
// the chain is correct. We work back through the checkpoints to find the latest that matches the
// actual data and work forwards from that one until either something fails or we reach the current
// head (then we save a checkpoint).
//
// I am wondering about pushing most of the logic and validation state management into
// the blockchain implementation. Along with a speculative `Fork() (Chain, error)`,
// `Commit(state) error` and `Drop() error` functions to allow handling multiple possible chains
// each peer will have a fork, and we will update the underlying Chains when we decide to commit
// a block. That is on Commit() of a block, any chain with a different block at that point should be dropped
// which may mean disconnecting from a peer sending bad blocks.
type p2pNode struct {
	options   *nodeOptions
	chain     blockchain.Chain
	peers     struct{} // @todo
	validator *ElectionValidator
}

// run the node!
func (n *p2pNode) Run(ctx context.Context) error {
	// To run the node we need to:
	// - validate our current chain (whether complete or not)
	// - if (validateOnly) return now.
	// - otherwise:
	//   - start our gRPC server (unless with-no-listener)
	//   - connect to any peers we have been given.
	//   - if no peers given and no listener, bail.
	//   - validate new blocks as they come in
	return nil
}

func (n *p2pNode) GetResult() *ElectionStats {
	return n.validator.state.GetResult()
}

func (n *p2pNode) GetBenchmarks() *ElectionBenchmarks {
	n.validator.state.GetResult()
	return n.validator.state.benchmarks
}
func (n *p2pNode) GetTimings() map[string]int {
	return n.validator.GetTimings()
}

type nodeOptions struct {
	electionId   ID
	listenAddr   string
	publicAddr   string
	seedPeers    []string
	maxPeers     int
	chainData    string //directory to store chain data
	validateOnly bool
}

type NodeOption interface {
	apply(*nodeOptions)
}

type optionFunc func(*nodeOptions)

func (f optionFunc) apply(opts *nodeOptions) {
	f(opts)
}

// now the options.

func WithSeedPeers(peers []string) NodeOption {
	return optionFunc(func(o *nodeOptions) {
		// we should de-dupe, but we won't do it here.
		o.seedPeers = append(o.seedPeers, peers...)
	})
}

func WithExternalAddr(addr string) NodeOption {
	return optionFunc(func(o *nodeOptions) {
		o.publicAddr = addr
	})
}

func WithListenAddr(addr string) NodeOption {
	return optionFunc(func(o *nodeOptions) {
		o.listenAddr = addr
	})
}

func WithDataDir(dir string) NodeOption {
	return optionFunc(func(o *nodeOptions) {
		o.chainData = dir
	})
}

// Only connect out, do not listen for inbound connections
// (only auditors go full mesh)
func WithNoListener() NodeOption {
	return optionFunc(func(o *nodeOptions) {
		o.listenAddr = ""
	})
}

func WithValidateOnly(validateOnly bool) NodeOption {
	return optionFunc(func(o *nodeOptions) {
		o.validateOnly = validateOnly
	})
}

// electionID is _required_ so it is a hard option
func Node(electionId ID, options ...NodeOption) (*p2pNode, error) {
	opts := &nodeOptions{
		electionId:   electionId,
		listenAddr:   "localhost:0",
		seedPeers:    []string{},
		chainData:    ".",
		validateOnly: false,
		// we cannot apply external address yet, if port was ephemeral,
		// so we instead leave it empty
	}
	for _, f := range options {
		f.apply(opts)
	}
	// now start the server.
	// first we validate the existing chain.
	// this means we need our stateful block validator for the election.
	validator := NewElectionValidator(electionId)
	// we need to open the chain for this election and validate it.
	chain, err := blockchain.Open(opts.chainData, opts.electionId, AstrisWorkLevel, validator)
	if err != nil {
		return nil, err
	}
	// we have a valid chain! That's great. now run the p2p server.
	return &p2pNode{
		options:   opts,
		chain:     chain,
		validator: validator,
		peers:     struct{}{}, // we need to do something with this list.
	}, nil

}

type canonicalJSON struct{}

// Encode the object in its canonical representation to the output stream given
func (c canonicalJSON) Encode(out io.Writer, v interface{}) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "")
	enc.SetEscapeHTML(false)
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var t interface{}
	err = json.Unmarshal(b, &t)
	if err != nil {
		return err
	}
	// t is map[string]interface instead of struct, so the keys will be sorted.
	return enc.Encode(t)
}

// Hash the object given in its canonical JSON representation
// the hash is SHA256
func (c canonicalJSON) Hash(b []byte, v interface{}) ([]byte, error) {
	h := sha256.New()
	if err := c.Encode(h, v); err != nil {
		return nil, err
	}
	return h.Sum(b), nil
}

func SliceToID(b []byte) (id ID) {
	for i := range id {
		id[i] = b[i]
	}
	return
}

// EncodeAndHash encodes the object in its canonical representation to the output
// stream and calculates the hash at the same time
func (c canonicalJSON) EncodeAndHash(out io.Writer, hash []byte, v interface{}) ([]byte, error) {
	h := sha256.New()
	m := io.MultiWriter(out, h)
	if err := c.Encode(m, v); err != nil {
		return nil, err
	}
	return h.Sum(hash), nil
}

func (c canonicalJSON) HashCheck(v interface{}, expected []byte) bool {
	actual, _ := c.Hash(nil, v)
	return bytes.Compare(actual, expected) == 0
}

// Canonical JSON encoding in astris.
//
// Basically the rules are, sort keys, minimal whitespace, add a
// trailing newline and follow the Go json.Encode rules.
//
// To create the canonical encoding:
//
//   - all objects have the keys sorted
//
//   - no extraneous whitespace (key spacing or indentation)
//
//   - no extraneous unicode escaping is done, except that the values
//     \u2028 (LINE SEPARATOR) and \u2029 (PARAGRAPH SEPARATOR) which
//     although technically allowed, are always escaped.
//
//   - big.Ints are encoded as base64url strings without padding
//
//   - Add a final trailing newline (0x0a) byte after the final `}`
//     (note that this is not after every object close)
//
//   - if in doubt, see the golang json encoder
//
//     NB this is important as the hashes will not match if canonical representation
//     if not used.
//
//     @TODO publish some canonical data with some edge cases for comparision
//     or create a typescript encoder that matches.
var CanonicalJSON = canonicalJSON{}

// a helper for json error messages/status responses in HTTP
// NB, not canonical json
func SimpleJSONResponse(wr http.ResponseWriter, code int, msg string) {
	obj := map[string]interface{}{
		"status":  http.StatusText(code),
		"message": msg,
		"code":    code,
	}
	wr.Header().Set("content-type", "application/json")
	wr.WriteHeader(code)
	enc := json.NewEncoder(wr)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(obj)
}
