# Benchmarks

To assess the viability of Astris I ran a number of timings using the software under different conditions.

The timings here, never include block creation as the PoW function can be tuned for difficulty as desired.
It is also not relevant during the audit process.

The biggest factor to timing is the choice of curve parameters. The bigger the curve, the longer it takes.

To keep things secure, I will use the `DH2048modp256` El Gamal params from RFC5114, as a "known" curve and
of the correct order for good security properties.

## Measurements

We have 2 dimensions to work with:

- `Nv` Number of Voters: `1000`,`10_000`, `100_000`, and `1_000_000`
- `Nk/Nt` Number of Trustees `Nt` and the required threshold for sharing `Nk`: `2/3`, `3/5`, `5/7`

Other factors:

- `Nc` number of candidates - affects the size of ZKPs and therefore the vote encryption and verification stages. All
  our benchmarks done with 10 candidates.

### Timing of Audit Process of the Stages

NB we benchmark the auditing process as the live creation process cryptographic functions are dwarfed by blockchain speed, so
it is most important to see that verification is possible both in realtime and much faster than realtime after the event.

#### `1000:2/3`

- Setup: `7ms`
- Voting: `26610ms`
- Talling: `74ms`
- Registration: `1247ms`
- Total: `XXXXms`

#### `10_000:2/3`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `100_000:2/3`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `1_000_000:2/3`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `1000:3/5`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `10_000:3/5`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `100_000:3/5`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `1_000_000:3/5`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `1000:5/7`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `10_000:5/7`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `100_000:5/7`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

#### `1_000_000:5/7`

- Setup: `XXXXms`
- Voting: `XXXXms`
- Talling: `XXXXms`
- Total: `XXXXms`

### Key Operation Timing

Timing Averages for key operations in the system, for a single participant, measured over the course
of running all the other benchmarks.

- Voter Registration: `XXXXms`
- Vote Encryption (and ZKP creation): `XXXXms`
- Vote Verification: `XXXXms`
- Partial Decryption: `XXXXms`
- Threshold Decryption: `XXXXms`

## Scripting

I need to automate as much of this as possible.

First we have a setup process to simulate the "authority" creating the election params

This is `authority simulate` command
