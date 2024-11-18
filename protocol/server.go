package protocol

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/thechriswalker/go-astris/astris"
	"github.com/thechriswalker/go-astris/blockchain"
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
	api astris.P2PNode

	UnimplementedAstrisV1Server
}

func (s *Server) seenPeer(ctx context.Context) error {
	if client, ok := peer.FromContext(ctx); ok {
		_ = client // do something with the client.
		// this is where we would perform some sort of blacklisting/whitelisting
		// for malicious clients
		if s.api.ShouldBlacklist(client.Addr) {
			return status.Error(codes.PermissionDenied, "Peer Blacklisted")
		}
		return nil
	} else {
		// let the system know this peer is alive
		s.api.PeerSeen(client.Addr)
		return status.Error(codes.Unavailable, "Client Address Unknown")
	}
}

// GetPeers allows us to send all our known peers for this election and annouce ones we discover
func (s *Server) PeerExchange(peer *Peer, stream AstrisV1_PeerExchangeServer) error {
	if err := s.seenPeer(stream.Context()); err != nil {
		return err
	}
	// @TODO implementation!
	return status.Error(codes.Unimplemented, "Not Implemented")
}

// RecieveBlocks gives us a channel to announce new blocks we validate
func (s *Server) RecvBlocks(empty *Empty, stream AstrisV1_RecvBlocksServer) error {
	if err := s.seenPeer(stream.Context()); err != nil {
		return err
	}
	return s.api.StreamNewBlocks(stream.Context(), func(blk *blockchain.BlockHeader) error {
		return stream.Send(toGRPCBlockHeader(blk))
	})
}

func toGRPCBlockHeader(blk *blockchain.BlockHeader) *BlockHeader {
	return &BlockHeader{
		PrevId:      blk.PrevID[:],
		PayloadHash: blk.PayloadHash[:],
		Timestamp:   blk.EpochSeconds,
		Proof:       blk.Proof,
		Depth:       blk.Depth,
		Kind:        uint32(blk.PayloadHint),
	}
}

func toAstrisHeader(blk *BlockHeader) (*blockchain.BlockHeader, error) {
	prevId, err := getIdFromBytes(blk.PrevId)
	if err != nil {
		return nil, err
	}
	payloadHash, err := getIdFromBytes(blk.PayloadHash)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid PayloadHash")
	}

	hdr := &blockchain.BlockHeader{
		PrevID:       prevId,
		EpochSeconds: blk.Timestamp,
		Proof:        blk.Proof,
		Depth:        blk.Depth,
		PayloadHash:  payloadHash,
		PayloadHint:  uint8(blk.Kind),
	}
	// we don't actually send the block id in the protocol, it is implicit
	hdr.ID = hdr.CalculateBlockID()
	return hdr, nil
}

func getIdFromBytes(b []byte) (id astris.ID, err error) {
	// make this a function...
	if len(b) != astris.IDSize {
		return id, status.Error(codes.InvalidArgument, "Invalid BlockID")
	}
	copy(id[:], b)
	return id, nil
}

// FetchBlock gets a block directly by hash
func (s *Server) GetBlock(ctx context.Context, b *BlockID) (*FullBlock, error) {
	if err := s.seenPeer(ctx); err != nil {
		return nil, err
	}
	id, err := getIdFromBytes(b.Id)
	if err != nil {
		return nil, err
	}
	blk, err := s.api.GetBlockWithPayload(id)
	if err != nil {
		if errors.Is(err, blockchain.ErrBlockMissing) {
			return nil, status.Error(codes.NotFound, "Block not found")
		}
		// something bad happened
		log.Err(err).
			Str("blkid", id.String()).
			Msg("Error fetching block GetBlock")
		return nil, status.Error(codes.Internal, "Something bad happened")
	}
	// we got the block, all good. turn it into a grpc block.
	full := &FullBlock{
		Header:  toGRPCBlockHeader(blk.Header),
		Payload: blk.Payload,
	}
	return full, nil
}

func (s *Server) AtDepth(ctx context.Context, d *Depth) (*BlockID, error) {
	if err := s.seenPeer(ctx); err != nil {
		return nil, err
	}
	blk, err := s.api.GetBlockHeaderAtDepth(d.Depth)
	if err != nil {
		if err != nil {
			if errors.Is(err, blockchain.ErrBlockMissing) {
				return nil, status.Error(codes.NotFound, "Block not found")
			}
			// something bad happened
			log.Err(err).
				Uint64("depth", d.Depth).
				Msg("Error fetching block AtDepth")
			return nil, status.Error(codes.Internal, "Something bad happened")
		}
	}
	// all good. return the ID
	return &BlockID{Id: blk.ID[:]}, nil
}

func (s *Server) FromBlock(b *BlockID, stream AstrisV1_FromBlockServer) error {
	if err := s.seenPeer(stream.Context()); err != nil {
		return err
	}
	id, err := getIdFromBytes(b.Id)
	if err != nil {
		return err
	}
	return s.api.StreamBlocksFromID(stream.Context(), id, func(blk *blockchain.Block) error {
		return stream.Send(&FullBlock{
			Header:  toGRPCBlockHeader(blk.Header),
			Payload: blk.Payload,
		})
	})
}

func (s *Server) PublishBlock(ctx context.Context, b *FullBlock) (*Acceptance, error) {
	if err := s.seenPeer(ctx); err != nil {
		return nil, err
	}
	hdr, err := toAstrisHeader(b.Header)
	if err != nil {
		return nil, err
	}
	// create the block as we need it
	accepted, err := s.api.NewBlock(&blockchain.Block{
		Header:  hdr,
		Payload: b.Payload,
	})
	if err != nil {
		// something bad happened
		log.Err(err).Msg("Error Validating New Block")
		return nil, status.Error(codes.Internal, "Something bad happened")
	}
	return &Acceptance{Accepted: accepted}, nil
}

var _ AstrisV1Server = (*Server)(nil)
