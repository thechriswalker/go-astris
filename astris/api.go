package astris

import "net"

type Node interface {
	// update our known peers list that we have a sighting of this peer
	PeerSeen(addr net.Addr)
}
