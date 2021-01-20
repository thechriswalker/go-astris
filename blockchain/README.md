# The blockchain

This is a simple blockchain.

Each block can have an arbitrary payload up to 2^16 bytes.

A SHA256 hash of the payload is stored with it as that is used in the header.

The ID of a block uses the ID of the previous block and the payload hash and the timestamp. So the ID is tied to the previous block and its payload.

The Proof of Work is simple HashCash style algorithm over the 2 IDs so can be calculated independently of the Block IDs.

The Genesis block has a previous ID set to all ZEROs.

We store the blocks in sqlite.

The network services should maintain speculative blocks and have a way to remove blocks from a chain to replace the chain with the new "longer" version.
