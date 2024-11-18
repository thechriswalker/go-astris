package voter

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb/v3"
	big "github.com/ncw/gmp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/thechriswalker/go-astris/astris"
	"github.com/thechriswalker/go-astris/blockchain"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
	"github.com/thechriswalker/go-astris/crypto/random"
)

// this is used to make array types.
const nCandidates = 10

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var dataDir string
	var electionIdStr string
	var nVoters int

	var voteCmd = &cobra.Command{
		Use:   "voter",
		Short: "Voter Commands",
	}

	rootCmd.AddCommand(voteCmd)

	var simulateCmd = &cobra.Command{
		Use:   "simulate",
		Short: "Simulate Voting",
		Long:  "Registers and Votes for a simluated election",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Starting Voter Simulation")
			var electionId astris.ID

			err := electionId.FromString(electionIdStr)
			if err != nil {
				log.Fatal().Err(err).Str("election", electionIdStr).Msg("given election id was not valid")
			}

			validator := astris.NewElectionValidator(electionId)
			validator.LooseMode = true // speed things up.
			log.Info().Msg("Opening and validating blockchain...")
			chain, err := blockchain.Open(dataDir, electionId, astris.AstrisWorkLevel, validator)
			if err != nil {
				panic(err)
			}
			// we should have the genesis block, so we can get the params.
			system := validator.System()
			reg := loadRegistrar(dataDir, system) // find the registrar data. so we can simulate registration

			// allocate upfront
			voters := make([]*Voter, nVoters)

			regAvg := &astris.Average{}
			avgVoteCast := &astris.Average{}

			// find the start time of the voter registration
			genisisBlk, _ := chain.Payload(electionId)
			var genisis astris.PayloadElectionSetup
			json.Unmarshal(genisisBlk, &genisis)
			regStart, _ := genisis.Timing.VoterRegistration.Opens.ToTime(genisis.Timing.Timezone)
			castStart, _ := genisis.Timing.VoteCasting.Opens.ToTime(genisis.Timing.Timezone)
			var timestamp uint32 = uint32(regStart.Unix())
			// we need progress for the voting as it takes a LONG TIME.
			// but more important probably is the ability to "RESUME".
			// that means saving voter information. given the size we will store it in
			// directories based on the voter hash.
			// double segments mean 255 direcrtories then 255 directories then voters, by hash
			// this is OK, because we can create the hash deterministically from the voters.
			// to find where we are, we will introspect the validation.

			head, err := chain.Head()
			if err != nil {
				panic(err)
			}
			isRegistration := true
			resumeFrom := 0

			// setup of the election takes:
			// - 1 block for the genesis
			// - NumTrustees blocks for shares
			// - NumTrustees blocks for trustee public
			setupBlocks := 1 + 2*uint64(len(genisis.Trustees))

			switch head.PayloadHint {
			case uint8(astris.HintTrusteePublic): // we just finished the trustee bit, no registrations
				isRegistration = true
				resumeFrom = 0
			case uint8(astris.HintVoterReg): // could be all registred or partial.
				// assume the happy path. 11 blocks for setup, so depth-11 voters registered.
				last := head.Depth - setupBlocks
				if last == uint64(nVoters) {
					// all done.
					isRegistration = false
					resumeFrom = 0
					log.Info().Msg("Resuming from the start of the casting phase")
				} else {
					isRegistration = true
					resumeFrom = int(last + 1)
					log.Info().Uint64("last", last).Msg("Resuming the registration phase")
				}
			case uint8(astris.HintBallot):
				// assume the happy path.
				last := head.Depth - setupBlocks - uint64(nVoters)
				if last == uint64(nVoters) {
					// all done.
					log.Info().Msg("All voter registration and casting is complete")
					return
				} else {
					isRegistration = false
					resumeFrom = int(last + 1)
					log.Info().Uint64("last", last).Msg("Resuming the casting phase")
				}
			default:
				panic("doesn't look like the correct part of the chain")
			}

			if isRegistration {
				// pass one.
				log.Info().Int("count", nVoters).Msg("Performing Voter Registration")
				bar := MaybeProgress(nVoters)
				bar.Start()
				for i := range voters {
					timestamp++
					if i < resumeFrom {
						voters[i] = loadVoterFromFile(dataDir, i+1, system)
						bar.Increment()
						continue
					}
					s := time.Now()
					// register a voter (and choose how they voted!)
					v := &Voter{
						ID: fmt.Sprintf("Voter[%d]", i+1),

						KeyPair: elgamal.GenerateKeyPair(system),
						Ballot:  makeVotes(),
					}
					v.Hash = sha256Hex(v.ID)
					writeVoterObjectToFile(dataDir, v)
					voters[i] = v
					// create the payload
					payload := &astris.PayloadVoterRegistration{
						VoterId:    v.ID,
						SigningKey: v.KeyPair.Public(),
					}

					payload.RSignature = reg.Secret().CreateSignature(payload.RSigMessage())
					payload.VSignature = v.KeyPair.Secret().CreateSignature(payload.VSigMessage())

					regAvg.Add(time.Since(s))

					// make the block!
					blk, err := astris.NewBlockBase(astris.HintVoterReg, payload)
					if err != nil {
						panic(err)
					}
					// mint it.
					if err := chain.Mint(blk, validator.WorkLevel(), timestamp); err != nil {
						panic(err)
					}
					// boom.
					bar.Increment()
				}
				bar.Finish()
				resumeFrom = 0
			} else {
				// we need to load all the voter data.
				log.Info().Int("count", nVoters).Msg("Loading Voter Registration Data")
				bar := MaybeProgress(nVoters)
				bar.Start()
				for i := range voters {
					voters[i] = loadVoterFromFile(dataDir, i+1, system)
					bar.Increment()
				}
				bar.Finish()
			}

			timestamp = uint32(castStart.Unix()) // start of vote casting

			pk := validator.ElectionPublicKey()

			// no need to allocate new each time.
			var cipherSum *elgamal.CipherText
			randomnessSum := big.NewInt(0)

			ciphers := make([]*elgamal.CipherText, nCandidates)
			proofs := make([]elgamal.ZKPOr, nCandidates)

			optionsCache := elgamal.NewPlaintextOptionsCache(system)
			zeroOrOne := optionsCache.GetOptions(1)  // max 1 or 0 in a single vote
			sumOptions := optionsCache.GetOptions(1) // max 1 vote

			log.Info().Int("count", nVoters).Msg("Performing Vote Casting")
			bar := MaybeProgress(nVoters)
			bar.Start()

			// now do the actual voting
			localTally := map[int]uint64{}
			for i, voter := range voters {
				timestamp++
				if i < resumeFrom {
					bar.Increment()
					continue
				}
				s := time.Now()
				// encrypt the voteswith the election public key
				nv := 0                   // num votes cast
				cipherSum = nil           // homomorphic ciphertext sum
				randomnessSum.SetInt64(0) // sum of the randomness
				for c := range ciphers {
					v := voter.Ballot[c]
					nv += v
					localTally[c] += uint64(v)
					// pick randomness
					rnd := random.Int(system.Q)
					randomnessSum.Add(randomnessSum, rnd)
					randomnessSum.Mod(randomnessSum, system.Q)

					// encrypt
					ciphers[c] = pk.Encrypt(zeroOrOne[v], rnd)
					cipherSum = cipherSum.Mul(system, ciphers[c])

					// create proofs
					proofs[c] = elgamal.ProveEncryption(pk, ciphers[c], zeroOrOne, v, rnd, []byte(voter.Hash))
				}
				// now we have all the values
				// we can create the final proof
				payload := &astris.PayloadCastVote{
					VoterId: voter.ID,
					Votes:   ciphers,
					Proofs:  proofs,
					Proof:   elgamal.ProveEncryption(pk, cipherSum, sumOptions, nv, randomnessSum, []byte(voter.Hash)),
				}
				payload.Signature = voter.KeyPair.Secret().Sign(payload)

				avgVoteCast.Add(time.Since(s))

				// boom! now mint it!
				// make the block!
				blk, err := astris.NewBlockBase(astris.HintBallot, payload)
				if err != nil {
					panic(err)
				}
				// mint it.
				if err := chain.Mint(blk, validator.WorkLevel(), timestamp); err != nil {
					panic(err)
				}
				bar.Increment()
			}
			bar.Finish()
			fmt.Println("localTally", localTally)
			result := validator.GetResult()
			// output to stdout as JSON
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(result)
			log.Info().Int("avg_ms", regAvg.Milliseconds()).Msg("Average Voter Registration")
			log.Info().Int("avg_ms", avgVoteCast.Milliseconds()).Msg("Average Vote Casting")
		},
	}

	voteCmd.PersistentFlags().StringVar(&dataDir, "data-dir", ".", "Directory to store data")
	voteCmd.PersistentFlags().StringVar(&electionIdStr, "election-id", "", "The election ID")
	simulateCmd.Flags().IntVar(&nVoters, "num-voters", 10, " The number of voters to simulate")

	voteCmd.AddCommand(simulateCmd)
}

type maybeProgress struct {
	bar *pb.ProgressBar
}

func MaybeProgress(n int) *maybeProgress {
	mp := &maybeProgress{}
	if n > 1000 {
		mp.bar = pb.ProgressBarTemplate(`{{string . "prefix"}}{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{etime . }`).New(n)
		mp.bar.SetRefreshRate(time.Second)
	}
	return mp
}

func (mp *maybeProgress) Start() {
	if mp.bar != nil {
		mp.bar.Start()
	}
}
func (mp *maybeProgress) Increment() {
	if mp.bar != nil {
		mp.bar.Increment()
	}
}

func (mp *maybeProgress) Finish() {
	if mp.bar != nil {
		mp.bar.Finish()
	}
}

type Voter struct {
	ID      string
	Hash    string
	KeyPair *elgamal.KeyPair
	Ballot  [nCandidates]int
}

// our simulated election has [nCandidates] candidates.
type Ballot [nCandidates]int

func makeVotes() (b Ballot) {
	// number of votes to make is 0,1,2
	// most people will vote twice, less once
	// and less 0 times.
	// say 0.1% is 0
	// 33.3% is 1
	// 66.6% is 2
	// so pick a number...
	var numVotes int
	rnd := rand.Intn(1000)
	switch {
	case rnd == 0:
		numVotes = 0
	// case rnd < 333:
	// 	numVotes = 1
	// default:
	// 	numVotes = 2
	default:
		numVotes = 1
	}
	choices := map[int]int{}
	for v := 0; v < numVotes; v++ {
		for {
			c := rand.Intn(nCandidates)
			if _, ok := choices[c]; !ok {
				choices[c] = 1
				break
			}
		}
	}
	for v := range b {
		b[v] = choices[v]
	}
	return
}

func loadRegistrar(dir string, system *elgamal.System) *elgamal.KeyPair {
	buf, err := os.ReadFile(filepath.Join(dir, "simulated-registrar.json"))
	if err != nil {
		panic(err)
	}
	kp := &elgamal.KeyPair{}
	err = json.Unmarshal(buf, kp)
	if err != nil {
		panic(err)
	}
	kp.Secret().System = system
	kp.Public().System = system
	return kp
}

func loadVoterFromFile(basedir string, idx int, system *elgamal.System) *Voter {
	id := fmt.Sprintf("Voter[%d]", idx)
	hash := sha256Hex(id)
	filename := filepath.Join(basedir, "voters", hash[0:2], hash[2:4], hash+"-voter.json")
	buf, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	v := &Voter{}
	err = json.Unmarshal(buf, v)
	if err != nil {
		panic(err)
	}
	v.KeyPair.Secret().System = system
	v.KeyPair.Public().System = system
	return v
}

func writeVoterObjectToFile(basedir string, v *Voter) {
	dir := filepath.Join(basedir, "voters", v.Hash[0:2], v.Hash[2:4])
	filename := filepath.Join(dir, v.Hash+"-voter.json")
	// ignore errors, panic later
	os.MkdirAll(dir, 0777)
	f, _ := os.Create(filename)
	defer f.Close()
	if err := astris.CanonicalJSON.Encode(f, v); err != nil {
		panic(err)
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
