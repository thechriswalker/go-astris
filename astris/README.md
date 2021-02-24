# the main process.

The "loop" here is fairly complex.

We keep a list of peers, ordered by a last_seen.

Everytime we see a message from a peer we update the last seen.

If they send us bad data we mark them as bad, we keep a list of bad peers.

The P2P loop runs as follows, one goroutine per peer:

- initiate a GRPC call to getpeers
- if the initial call does not error, start a goroutine collecting the peers and send them on down the channel to the main co-ordinator
- initiate a GRPC call to recvblocks
- if the initial call does not error, start a goroutine to receive the blocks
- if a number of bad peers or broken blocks arrive, sever the connections and add to blacklist
- for each block recieved:
  - if it is a valid candidate for a longer chain than we have, accept
    and request all intermediate blocks to confirm, then send on the channel to confirm with the co-ordinator.
  - if it is valid but not as deep, reject it but do not penalise the sender
  - if it is invalid, reject it and penalise the sender.
- if the peer is penalised enough, blacklist them.
- if we don't hear from a peer or cannot ping them, then end the goroutine, but do not blacklist them.

## notes

<!-- WHILE technically correct, the problem is in the message expansion of the encryption, which is massive. So discard this text...

Votes are stored for each candidate in a single integer.
We use a fixed space for each integer, so when we **add** them, provided the max
number of voters is less than the space for each candidate then everything is good.

This means we need to ensure we provide enough space in a candidates/voters trade-off

i.e. if we say max 255 voters, (`2^8 - 1`) then we need 8-bits per candidate and in a 64bit number we can have 8 candidates.

But for a national election we may have a magnitude of 100 million voters. which means we probably want 32bits for each candidate, so a 64bit vote would only be able to have 2 candidates.

Ideally we want to be able to support a good deal more than we would ever need.
A 32bit vote is 4,294,967,295 (4.2 billion) voters. That is half the planet, so to include everyone lets go 64bit, which should give us some time.

We have already gone past what our computers can easily handle with 1 candidate now, so we need to be able to support something bigger than that.

So we will make the number of bits per candidate (based on electorate size) and the number of candidates has to fit inside our block payload with space for the signatures.

The block can hold 2^16 bytes (64MiB) by convention.
This means the vote size must be less than that.
Let BITS_PER_CANDIDATE be the smallest power of 2 greater than the size of the electorate. This
is fixed and non-negiotable. So we have a limit on the number of candidates possible

VOTE_SIZE = BITS_PER_CANDIDATE \* CANDIDATES < MAX_BLOCK_PAYLOAD

The block payload is 64MiB so with an electorate of 100 million we need 27 bits per candidate (`2^26 = 67,108,864` and `2^27 = 134,217,728` so actually we could up to 134million votes with 27 bits).

So the max candidates has to fit in 2^16 (with some room for signatures, say 1024bits
Rearragning the equation:

MAX_CANDIDATES = ROUND_DOWN((MAX_BLOCK_PAYLOAD_BITS - 1024) / 27) = 19380

So our blockchain can support nearly 20000 candidates for a 130million voter election. I think that should be enough.
 -->

We plan to use dBFV distributed homomorphic encryption. depending on the parameters we can have more or less candidates and voters.

The default "small" parameters of the library I have chosen allow up to 4096 candidates and 65,929,217 voters. This gives a "vote" size of
131,077 bytes for the ciphertext.
