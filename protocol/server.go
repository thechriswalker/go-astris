package protocol

import (
	"context"

	"github.com/thechriswalker/go-astris/astris"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Server is the server implementation of the Astris Protocol
// That is, the thing that other nodes connect to to get data.
// The Client is what "this" node connects to for data.
// So, the Server is unconcerned by the peers that are connected
// to it,
type Server struct {
	api astris.Node

	UnimplementedAstrisNodeServer
}

// Obtain a list of peers to connect to, of course we
// must seed some peers first...
//	GetPeers(*ElectionID, AstrisNode_GetPeersServer) error
// this is the connection we open to receive blocks
// we only have a recieve function. If other peers
// want blocks from us, they must call RecieveBlocks
// to open the channel
// We pass an ElectionId and assume all blocks on the channel
// are for the given election. If the server does not support
// this election, they SHOULD end the stream immediately
//	RecieveBlocks(*ElectionID, AstrisNode_RecieveBlocksServer) error
// recieve a specific block from the peer, if they have it
// this can be used to populate your chain
// this is irrespective of ElectionID as all the blocks are uniquely
// addressed.
//	FetchBlock(context.Context, *BlockID) (*FullBlock, error)
// get the block the peer considers to be the top of the chain
// a given chain may have more than one head as we speculate, but this
// is the block the server considers the "head". In Bitcoin this
// is usually 6 blocks back from the whatever the top of the chain is.
//	Head(context.Context, *ElectionID) (*BlockHeader, error)

func (s *Server) seenPeer(ctx context.Context) error {
	if client, ok := peer.FromContext(ctx); ok {
		s.api.PeerSeen(client.Addr)
		return nil
	} else {
		return status.Error(codes.Unavailable, "Client Address Unknown")
	}
}

// GetPeers allows us to send all our known peers for this election and annouce ones we discover
func (s *Server) GetPeers(electionID *ElectionID, stream AstrisNode_GetPeersServer) error {
	if err := s.seenPeer(stream.Context()); err != nil {
		return err
	}
	// @TODO implementation!
	return status.Error(codes.Unimplemented, "Not Implemented")
}

// RecieveBlocks gives us a channel to announce new blocks we validate
func (s *Server) RecieveBlocks(electionID *ElectionID, stream AstrisNode_RecieveBlocksServer) error {
	if err := s.seenPeer(stream.Context()); err != nil {
		return err
	}
	// @TODO implementation!
	return status.Error(codes.Unimplemented, "Not Implemented")
}

// FetchBlock gets a block directlty by hash
func (s *Server) FetchBlock(ctx context.Context, blockID *BlockID) (*FullBlock, error) {
	if err := s.seenPeer(ctx); err != nil {
		return nil, err
	}
	// @TODO implementation!
	return nil, status.Error(codes.Unimplemented, "Not Implemented")
}

// Head returns the last confirmed block in our chain.
func (s *Server) Head(ctx context.Context, electionID *ElectionID) (*BlockHeader, error) {
	if err := s.seenPeer(ctx); err != nil {
		return nil, err
	}
	// @TODO implementation!
	return nil, status.Error(codes.Unimplemented, "Not Implemented")
}

var _ AstrisNodeServer = (*Server)(nil)
