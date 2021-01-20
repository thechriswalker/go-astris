# Astris eVoting

This is a proof of concept software implementation of the Astris eVoting scheme.

Astris aims to reduce the amount of trust required of a system by it's users. That is, the users should only have to trust a minimum of entities and therefore have an inversely proportinal amount of confidence in the system.

Astris defines a scheme for a blockchain based system where the integrity of each step is controlled by a shared ruleset, enforced by the software and trusted by consensus. It uses a private chain and all data is in the blocks.

Features:

- [ ] SQLite backed local persistence
- [ ] Peer discovery
- [ ] Election initialisation
- [ ] Blockchain based election integrity, with simple Proof Of Work
- [ ] Blockchain consensus amongst peers
- [ ] Sample `EligibilityAuthority` required to prove eligibility of a voter.
- [ ] Peer consensus for chain validation.
- [ ] Offline full election verification
