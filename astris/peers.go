package astris

import (
	"sync"
	"time"
)

// PeerList is what our node keeps track of and manages the lifecycles of.
type PeerList struct {
	// mutex guarded map of Peers keyed on their addr
	mtx   sync.RWMutex
	peers map[string]*Peer
}

// Len returns the number of peers in the list
func (pl *PeerList) Len() int {
	pl.mtx.RLock()
	defer pl.mtx.RUnlock()
	return len(pl.peers)
}

// Get a single peer from the list and whether
// it was found
func (pl *PeerList) Get(addr string) (*Peer, bool) {
	pl.mtx.RLock()
	defer pl.mtx.RUnlock()
	p, ok := pl.peers[addr]
	return p, ok
}

// GetList returns an array of the peer addresses
func (pl *PeerList) GetList() []string {
	pl.mtx.RLock()
	defer pl.mtx.RUnlock()
	list := make([]string, len(pl.peers))
	i := 0
	for addr := range pl.peers {
		list[i] = addr
		i++
	}
	return list
}

// Peer represents a Node in our P2P network
type Peer struct {
	host     string
	lastSeen time.Time // if time.IsZero(peer.lastSeen) then we have never tried this peer
}

// NewPeer initialises a peer
func NewPeer(addr string) *Peer {
	return &Peer{
		addr:     addr,
		lastSeen: time.Time{},
	}
}
