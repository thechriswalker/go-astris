package authority

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-input"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
	"github.com/thechriswalker/puid"
)

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var authorityConfigFile string
	var authorityCmd = &cobra.Command{
		Use:   "authority",
		Short: "Election Setup Authority",
		Long:  "Allows creation of an election and the genesis block",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Starting Election Authority Process")
			// we store the blossoming election config in a file.
			setup := &ElectionSetup{}
			f, err := os.Open(authorityConfigFile)
			if os.IsNotExist(err) {
				log.Debug().
					Str("file", authorityConfigFile).
					Msg("Setup data file missing")
				log.Info().Msg("No Setup Data File: Starting New Election")
			} else {
				// decode waht we have so far...
				err := json.NewDecoder(f).Decode(setup)
				if err != nil {
					log.Error().
						Str("file", authorityConfigFile).
						Str("error", err.Error()).
						Msg("Failed to parse setup data. Not valid JSON?")
					log.Warn().Msg("Continuing now will overwrite the setup data.")
				} else {
					log.Info().
						Str("file", authorityConfigFile).
						Msg("Loaded setup data")
				}
				f.Close()
			}

			save := func() {
				f, err := os.Create(authorityConfigFile)
				if err != nil {
					log.Fatal().Msg("failed to save setup config")
				}
				defer f.Close()
				j := json.NewEncoder(f)
				j.SetEscapeHTML(false)
				j.SetIndent("", "  ") // make it pretty
				err = j.Encode(setup)
				if err != nil {
					log.Fatal().Msg("failed to save setup config")
				}
			}

			// now we go through the file and ask questions as needed to generate all the data.
			// if it is complete then we generate the electionId (from the hash of the setup data)
			// then we can check our blockchain for the election and add it if needed.
			ui := input.DefaultUI()
			// actually we will give the opportunity to change the data as well.
			setup.Name, err = ui.Ask(">>> Give the election a name", &input.Options{
				Default:      setup.Name,
				HideOrder:    true,
				ValidateFunc: notEmpty,
				Loop:         true,
			})
			if err != nil {
				log.Fatal().Msgf("Error reading input %s", err)
			}
			log.Info().Str("name", setup.Name).Msg("Election Name Confirmed")
			save()

			// now the election Encryption parameters. they are complicated, so we will choose
			// for the authority. The JSON can be changed.
			// NB we should validate the system parameters before accepting
			if setup.Params != nil {
				if err := setup.Params.Validate(); err != nil {
					log.Fatal().Msg("Stored ElGamal Parameters are invalid. Please remove and re-run")
				}
			} else {
				setup.Params = &elgamal.ThresholdSystem{System: elgamal.DH2048modp224()}
				log.Info().Msg("Using ElGamal Params from RFC5114 DH2048modp224.")
			}
			save()

			// @todo candidates
			// this whole thing should be a web-based form... much better UX and could generate the JSON I need for the genesis block.

			// @todo registrar

			// @todo timings

			// then trustees

			///// Params.L
			strL, err := ui.Ask(">>> How many trustees will there be?", &input.Options{
				HideOrder:    true,
				Default:      strconv.Itoa(max(0, setup.Params.L)),
				ValidateFunc: gt(0, -1),
				Loop:         true,
			})
			if err != nil {
				log.Fatal().Msgf("Error reading input: %s", err)
			}

			setup.Params.L, err = strconv.Atoi(strL)
			if err != nil {
				log.Fatal().Msgf("invalid value for number of trustees: %s", strL)
			}
			save()

			strTplus1, err := ui.Ask(">>> How many trustees should be required to decrypt data?", &input.Options{
				HideOrder: true,
				Default:   strconv.Itoa(max(1, setup.Params.T+1)),
				// 1 is a pathological case, but it still works.
				// L is the maximum, requiring everyone to participate
				ValidateFunc: gt(1, setup.Params.L),
				Loop:         true,
			})
			if err != nil {
				log.Fatal().Msgf("Error reading input: %s", err)
			}

			tPlus1, err := strconv.Atoi(strTplus1)
			if err != nil {
				log.Fatal().Msgf("invalid value for number of required trustees: %s", strTplus1)
			}
			setup.Params.T = tPlus1 - 1
			save()

			// now we must validate the trustees.
			// we should have a slice of Params.L trustees.
			if setup.Trustees == nil {
				setup.Trustees = make([]*TrusteeSetup, 0, setup.Params.L)
			} else {
				if len(setup.Trustees) > setup.Params.L {
					log.Fatal().Msgf("Too many trustees (expecting %d, found %d)", setup.Params.L, len(setup.Trustees))
				}
			}
			// now go through all L
			l := len(setup.Trustees)
			tid := puid.WithPrefixByte('t')
			for i := 0; i < setup.Params.L; i++ {
				if i >= l {
					setup.Trustees = append(setup.Trustees, &TrusteeSetup{
						TrusteeID: tid.New(),
						Name:      fmt.Sprintf("Trustee: %d", i+1),
					})
				}
				t := setup.Trustees[i]
				log.Info().Int("index", i).Str("id", t.TrusteeID).Msg("")
				name, err := ui.Ask(fmt.Sprintf(">>> Name for Trustee %d", i+1), &input.Options{
					HideOrder:    true,
					Default:      t.Name,
					ValidateFunc: notEmpty,
					Loop:         true,
				})
				if err != nil {
					log.Fatal().Msgf("Error reading input: %s", err)
				}
				t.Name = name
				// @todo ask for next stage info or skip
				save()
			}

		},
	}
	authorityCmd.Flags().StringVar(
		&authorityConfigFile,
		"config",
		"authority-config.json",
		"The file to store authority data",
	)
	rootCmd.AddCommand(authorityCmd)
}

func notEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("Empty String Not Allowed")
	}
	return nil
}

func gt(min, max int) func(s string) error {
	return func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		if max > min {
			if n < min || n > max {
				return fmt.Errorf("Must provide an integer <= %d and >= %d", min, max)
			}
		} else {
			if n < min {
				return fmt.Errorf("Must provide an integer <= %d", min)
			}
		}
		return nil
	}
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

// ElectionSetup this is want is loaded into the config file
// We can only add so much at a time, we need input from the previous
type ElectionSetup struct {
	Name       string                   `json:"name"`
	Params     *elgamal.ThresholdSystem `json:"encryptionSharedParams"`
	Candidates []*CandidateSetup        `json:"candidates"`
	Trustees   []*TrusteeSetup          `json:"trustees"`
	Registrar  *RegistrarSetup          `json:"registrar"`
	Timing     *TimingInfo              `json:"timing"`
}

type CandidateSetup struct {
	CandidateID string `json:"candidateId"`
	Name        string `json:"name"`
}

type TrusteeSetup struct {
	TrusteeID  string                    `json:"trusteeId"`
	Name       string                    `json:"name"`
	SigKey     *elgamal.PublicKey        `json:"verificationKey"`
	SigProof   *elgamal.ProofOfKnowledge `json:"verificationProof"`
	ShardKey   *elgamal.PublicKey        `json:"shardPublicKey"`
	ShardProof *elgamal.ProofOfKnowledge `json:"shardProof"`
}

type RegistrarSetup struct {
	RegistrarID     string                    `json:"registrarId"`
	Name            string                    `json:"name"`
	SigKey          *elgamal.PublicKey        `json:"verificationKey"`
	SigProof        *elgamal.ProofOfKnowledge `json:"verificationProof"`
	DataURL         string                    `json:"eligibilityDataURL"`
	DataHash        string                    `json:"eligibilityDataHash"`
	RegistrationURL string                    `json:"registrationURL"`
}

type TimeBounds struct {
	Opens  time.Time `json:"opens"`
	Closes time.Time `json:"closes"`
}

type TimingInfo struct {
	ParameterConfirmation *TimeBounds `json:"parameterConfirmation"`
	VoterRegistration     *TimeBounds `json:"voterRegistration"`
	VoteCasting           *TimeBounds `json:"voteCasting"`
	TallyDecryption       *TimeBounds `json:"tallyDecryption"`
}
