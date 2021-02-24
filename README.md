# Astris eVoting

This is a proof of concept software implementation of the Astris eVoting scheme.

Astris aims to reduce the amount of trust required of a system by it's users. That is, the users should only have to trust a minimum of entities and therefore have an inversely proportional amount of confidence in the system.

Astris defines a scheme for a blockchain based system where the integrity of each step is controlled by a shared ruleset, enforced by the software and trusted by consensus. It uses a private chain and all data is in the blocks.

Features:

- [ ] SQLite backed local persistence
- [ ] Peer discovery
- [ ] Election initialisation
- [ ] Blockchain based election integrity, with simple Proof Of Work
- [ ] Blockchain consensus amongst peers.
- [ ] Sample `EligibilityAuthority` required to prove eligibility of a voter.
- [ ] Peer consensus for chain validation.
- [ ] Offline full election verification.

## Ideas not implemented

This is a proof of concept and will require more features to be fully "production-ready".

Here is a list I created while writing the software:

- Automatic NAT/Firewall detection and use of STUN/TURN to enable behind-NAT comms. For now the user must, have explicit port forwarding and firewall rules.
- User accessible peer and/or persistent peer black/whitelist. For now then blacklist is in-memory only
- Fully dynamic BFV parameters for homomorphic encryption. This demo chooses a parameter set suitable for small elections and minimizes vote size.

## Astris voting protocol

### Initialisation: Election Parameters.

The "tallying authorities" combine to create a multi-part key for the threshold homomorphic verifiable encryption scheme, and the parameters.
We will have all the data for the "members" of the group along with any metadata for each one that we require (name, ID in the scheme, etc...)

- `tPk` the public key for the election
- `tSk[i]` the secret keys, one for each authority.
- `tPm` the parameters, which will be chosen wrt the number of candidates and voters.

At least one of the tallying authorities must be honest for the security of the election to hold.

Then we create the list of candidates `C[i]` Where each candidate has an integer identifier and a human readable name.

- e.g. `C = ["alice", "bob", "chris", "denise", "eve"]` where the index in the list +1 is the identifier, `0` is the "abstain" index

We also define the "eligibility authority" which will confirm that a use is eligible to vote.

This is done by election organiser accepting a public key from the authority. The Authority keeps and does not reveal the secret key.
An eligibility server's duty is to provide a digital signature over the vote. Instead of signing the actual vote data (so it never has access to it)
Instead a SHA256 sum of the vote data is signed, along with a token representing the voter.
The exact method for authenticating with the eligibility server is not yet defined, and will probably assume a web interface for flexibility of implementation.
The elgibility URL must be a URL that when adding the vote hash `?election=XXX&vote_hash=XXXX` (or `vote_hash=novote` to provide a no-vote ) will present a webpage which allows the user to authenticate and
the server will return a token which can be used to cast the vote with that hash and will be associated with the given voter. This will likely be done with an iframe and postMessage

- `ePk` the public key of the eligibility authority
- `eUrl` the url to the system for signing votes.

The election timing information:

- `open` - the date-time of when the election will start accepting votes.
- `close` - the date-time of when the election will stop accepting votes.

Then some human readable metadata:

- `name` The name of the election e.g. `2020 US Presidential Election`
- `description` a description of the election

This becomes the data for the genesis block of this election.
Note that the data will not be in JSON, but a binary encoding, which will likely not
have the field names. We are only aiming for a Golang implementation, so interoperability, while nice, is not required.

```json
{
  "kind":"initialisation",
  "tally": {
    "publicKey": "<binary>",
    "params": { "N": ... dBFV parameters },
  },
  "eligibility": {
    "publicKey": "<binary>",
    "URL": "https://eligibility.astris.org/confirm",
  },
  "candidates": ["alice", "bob", "chris", "denise", "eve"],
  "opens": "2020-01-01T00:00:00Z",
  "closes": "2020-01-04T00:00:00Z",
  "name": "Some election",
  "description": "an election using astris"
}
```

This payload creates a block for the start of the election.
Although the ElectionID will be the hash of this block and will be published when the election is set up, so the parameters cannot be changed.
At any point, NoVote blocks can be added to the chain to deepen it, by a given voter. It may be that implementors limit the number of NoVote blocks from a given voter.

From the "opens" date of the election voters can vote.
The process is:

- verify the election details
- choose your candidate.
- create the cyphertext (from the tally public key and params).
- generate a ZKP that the ciphertext contains at most a single vote for a single candidate (or abstention)
- hash the ciphertext SHA256
- go to `eligibility.URL` with extra params: `electionId=<election id>` and `vote_hash=<ciphertext SHA>` and allow the eligibility server to authenticate you
  and give you a `token` and a `voter_id` to identify your vote.
- your ballot to cast is now:

```json
{
    "kind": "vote",
    "election_id": "...",
    "vote": {
        "cast": "<binary>",
        "zkp": "<binary>",
    },
    "eligibility": {
        "token": "<token>",
        "voter": "<id from eligibility server"
    }
}
```

Any astris node receiving this vote in a block would verify the block first (obviously) and then vertifiy the contents.


 - Check the vote zkp is valid for the vote cast.
 - Check that the token is a valid signature over the vote hash and voter id, using the elgibility public key.

If anything is amiss then we reject the block.
A server may have more than one vote in a block.

Any server can add a `noop` block at the rate of X per minute on the chain. That is the timestamp of the block must be more than X time after
the timestamp on the previous `noop` block. These blocks have a fixed payload: the 4 bytes `[]byte("noop")`. The intent is to be able to keep the chain growing without
votes being cast.

Once the `closes` time is past the tally can be created, each node will perform this task, as a node receiving a tally will want to
make sure the tally is valid by checking the rest of the previous blocks.

Any server completing the tally my post it to the chain with a payload:

```json
{
    "kind":"tally",
    "tally":"<binary>"
}
```

Once the tally has been added, the multi-party decrypt can happen, with each authority server posting it's partial decryption and the ZKP of that partial decryption.

```json
{
    "kind":"partial_result",
    "part_id": "id of partial server",
    "partial_decrypt": "<binary>",
    "partial_zkp": "<binary>"
}
```


