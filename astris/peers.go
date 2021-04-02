package astris

import (
	"context"
	"time"

	"github.com/thechriswalker/go-astris/blockchain"
)

// PeerConnection represents a Node in our P2P network
// that _we_ are talking to. other nodes can connect to us if they want
// but this is the node we are connecting to.
type PeerConnection struct {
	host      string
	ctx       context.Context
	lastSeen  time.Time        // if time.IsZero(peer.lastSeen) then we have never tried this peer
	badBlocks int              // the number of bad blocks we have recieved
	recvPeers chan string      // this is the channel we send newly received peer addresses on
	chain     blockchain.Chain // a reference to the blockchain so we can build it further if needed
}
