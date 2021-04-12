package trustee

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/thechriswalker/go-astris/astris"
	"github.com/thechriswalker/go-astris/blockchain"
	"github.com/thechriswalker/go-astris/crypto"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
)

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var trusteeCmd = &cobra.Command{
		Use:   "trustee",
		Short: "Trustee Commands",
	}

	rootCmd.AddCommand(trusteeCmd)
	var dataDir string
	var electionIdStr string

	var tallyCmd = &cobra.Command{
		Use:   "simulate",
		Short: "Partial Decrypt a tally for the simulation",
		Run: func(cmd *cobra.Command, args []string) {
			// add up, decrypt.
			log.Info().Msg("Starting Voter Simulation")
			var electionId astris.ID

			err := electionId.FromString(electionIdStr)
			if err != nil {
				log.Fatal().Err(err).Str("election", electionIdStr).Msg("given election id was not valid")
			}

			validator := astris.NewElectionValidator(electionId)
			// speed this up for the simulation
			validator.LooseMode = true
			chain, err := blockchain.Open(dataDir, electionId, astris.AstrisWorkLevel, validator)

			// load all the trustee data.
			trustees := loadTrustees(dataDir, validator)

			var timestamp uint32 = 1617490801

			encTally := validator.GetLocalTally()
			//fmt.Printf("%#v\n", encTally)
			for i, t := range trustees {
				var _ = i
				payload := &astris.PayloadPartialTally{
					Index:     t.Index,
					Decrypted: make(crypto.BigIntSlice, len(encTally)),
					Proofs:    make([]*elgamal.ZKP, len(encTally)),
					Tallies:   encTally,
				}
				for c, ct := range encTally {
					payload.Decrypted[c] = t.Threshold.PartialDecrypt(ct)
					payload.Proofs[c] = elgamal.ProveDecryption(t.Threshold.ShardKey.Secret(), ct)
				}
				payload.Signature = t.Keys.Sig.Secret().Sign(payload)

				// mint the block
				blk, err := astris.NewBlockBase(astris.HintPartialTally, payload)
				if err != nil {
					panic(err)
				}
				if err := chain.Mint(blk, validator.WorkLevel(), timestamp); err != nil {
					panic(err)
				}
				timestamp += 30
			}
		},
	}
	trusteeCmd.PersistentFlags().StringVar(&electionIdStr, "election-id", "", "Election ID")
	trusteeCmd.PersistentFlags().StringVar(&dataDir, "data-dir", ".", "Directory holding data")
	trusteeCmd.AddCommand(tallyCmd)
}

type TrusteePrivate struct {
	Index     int
	Keys      *elgamal.DerivedKeys
	Threshold *elgamal.PrivateParticipant
}

func loadTrustees(dir string, validator *astris.ElectionValidator) []*TrusteePrivate {
	trustees := make([]*TrusteePrivate, validator.GetNumTrustees())
	sys := validator.System()
	for i := range trustees {
		index := i + 1
		filename := filepath.Join(dir, fmt.Sprintf("simulated-trustee-%d.json", index))
		f, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		dec := json.NewDecoder(f)
		t := &TrusteePrivate{}
		if err = dec.Decode(&t); err != nil {
			panic(err)
		}
		trustees[i] = t
		t.Keys.ReSystem(sys)
		t.Threshold.ShardKey.Public().System = sys
		t.Threshold.ShardKey.Secret().System = sys
		if err = f.Close(); err != nil {
			panic(err)
		}

	}
	return trustees
}
