package authority

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	big "github.com/ncw/gmp"

	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/thechriswalker/go-astris/astris"
	"github.com/thechriswalker/go-astris/blockchain"
	"github.com/thechriswalker/go-astris/crypto"
	"github.com/thechriswalker/go-astris/crypto/elgamal"
	"github.com/thechriswalker/go-astris/ui"
)

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var authorityConfigFile string
	var uiPort int16
	var openUI bool
	var authorityCmd = &cobra.Command{
		Use:   "authority",
		Short: "Election Setup Authority",
		Long:  "Allows creation of an election and the genesis block",
	}
	rootCmd.AddCommand(authorityCmd)

	interactiveCmd := &cobra.Command{
		Use:   "interactive",
		Short: "Run UI to setup election",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Starting Election Authority Process")
			// we store the blossoming election config in a file.
			setup := &astris.PayloadElectionSetup{}
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
			setup.Version = astris.AstrisProtocolVersion

			save := func() error {
				f, err := os.Create(authorityConfigFile)
				if err != nil {
					log.Error().Err(err).Msg("Error creating election file")
					return err
				}
				defer f.Close()

				setup.Version = astris.AstrisProtocolVersion
				h, err := astris.CanonicalJSON.EncodeAndHash(f, nil, setup)
				if err != nil {
					log.Error().Err(err).Msg("Error saving election file")
					return err
				}
				log.Debug().Str("hash", fmt.Sprintf("%02x", h)).Msg("Saved Election Config File")
				return nil
			}

			// start a web-server to ease the UI of the setup.
			// start it on a local ephemeral port (using localhost:0)
			// unless we are given a port.
			// and then we log the URL for the user to open.

			// to get an ephemeral port we have to get the listener directly.
			// so we can inspect it to find the port. But we might as well do
			// that anyway now.

			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", uiPort))
			if err != nil {
				log.Fatal().Err(err).Msg("Error starting webserver")
			}
			uiURL := "http://" + listener.Addr().String() + "/authority"
			log.Info().Str("url", uiURL).Msg("Started local webserver for UI")
			if openUI {
				err := open.Run(uiURL)
				if err != nil {
					log.Warn().Err(err).Msg("Failed to open system browser")
				}
			}
			mux := ui.AuthorityPage.Mux()

			// I think at this point we should load/save our JSON via a HTTP GET/PUT
			mux.HandleFunc("/authority/api/config.json", func(wr http.ResponseWriter, req *http.Request) {
				switch req.Method {
				case "GET":
					// return the content of the file
					req.Header.Add("content-type", "application/json;charset=utf-8")
					err := json.NewEncoder(wr).Encode(setup)
					if err != nil {
						// nothing to do but log it.
						log.Error().Err(err).Msg("Failed to encode authority config to HTTP response")
					}
				case "PUT":
					//
					// @TODO validate and save the content of the file
					// but for now, our validation will be parse.
					defer req.Body.Close()
					if err := json.NewDecoder(req.Body).Decode(&setup); err != nil {
						// err?
						log.Warn().Err(err).Msg("Bad input from UI")
						astris.SimpleJSONResponse(wr, http.StatusUnprocessableEntity, "Could not parse JSON input")
						return
					}
					if err := save(); err != nil {
						astris.SimpleJSONResponse(wr, http.StatusInternalServerError, "Could not save election config")
						return
					}
					// otherwise all good!
					astris.SimpleJSONResponse(wr, http.StatusOK, "Saved Configuration file")
				default:
					// unsupported method
					astris.SimpleJSONResponse(wr, http.StatusMethodNotAllowed, "Invalid HTTP Method for this endpoint")
				}
			})

			err = http.Serve(listener, mux)
			if err != nil {
				log.Fatal().Err(err).Msg("Error serving UI")
			}

			// // now we go through the file and ask questions as needed to generate all the data.
			// // if it is complete then we generate the electionId (from the hash of the setup data)
			// // then we can check our blockchain for the election and add it if needed.
			// ui := input.DefaultUI()
			// actually we will give the opportunity to change the data as well.
			// setup.Name, err = ui.Ask(">>> Give the election a name", &input.Options{
			// 	Default:      setup.Name,
			// 	HideOrder:    true,
			// 	ValidateFunc: notEmpty,
			// 	Loop:         true,
			// })
			// if err != nil {
			// 	log.Fatal().Msgf("Error reading input %s", err)
			// }
			// log.Info().Str("name", setup.Name).Msg("Election Name Confirmed")
			// save()

			// // now the election Encryption parameters. they are complicated, so we will choose
			// // for the authority. The JSON can be changed.
			// // NB we should validate the system parameters before accepting
			// if setup.Params != nil {
			// 	if err := setup.Params.Validate(); err != nil {
			// 		log.Fatal().Msg("Stored ElGamal Parameters are invalid. Please remove and re-run")
			// 	}
			// } else {
			// 	setup.Params = &elgamal.ThresholdSystem{System: elgamal.DH2048modp224()}
			// 	log.Info().Msg("Using ElGamal Params from RFC5114 DH2048modp224.")
			// }
			// save()

			// // @todo candidates
			// // this whole thing should be a web-based form... much better UX and could generate the JSON I need for the genesis block.

			// // @todo registrar

			// // @todo timings

			// // then trustees

			// ///// Params.L
			// strL, err := ui.Ask(">>> How many trustees will there be?", &input.Options{
			// 	HideOrder:    true,
			// 	Default:      strconv.Itoa(max(0, setup.Params.L)),
			// 	ValidateFunc: gt(0, -1),
			// 	Loop:         true,
			// })
			// if err != nil {
			// 	log.Fatal().Msgf("Error reading input: %s", err)
			// }

			// setup.Params.L, err = strconv.Atoi(strL)
			// if err != nil {
			// 	log.Fatal().Msgf("invalid value for number of trustees: %s", strL)
			// }
			// save()

			// strTplus1, err := ui.Ask(">>> How many trustees should be required to decrypt data?", &input.Options{
			// 	HideOrder: true,
			// 	Default:   strconv.Itoa(max(1, setup.Params.T+1)),
			// 	// 1 is a pathological case, but it still works.
			// 	// L is the maximum, requiring everyone to participate
			// 	ValidateFunc: gt(1, setup.Params.L),
			// 	Loop:         true,
			// })
			// if err != nil {
			// 	log.Fatal().Msgf("Error reading input: %s", err)
			// }

			// tPlus1, err := strconv.Atoi(strTplus1)
			// if err != nil {
			// 	log.Fatal().Msgf("invalid value for number of required trustees: %s", strTplus1)
			// }
			// setup.Params.T = tPlus1 - 1
			// save()

			// // now we must validate the trustees.
			// // we should have a slice of Params.L trustees.
			// if setup.Trustees == nil {
			// 	setup.Trustees = make([]*TrusteeSetup, 0, setup.Params.L)
			// } else {
			// 	if len(setup.Trustees) > setup.Params.L {
			// 		log.Fatal().Msgf("Too many trustees (expecting %d, found %d)", setup.Params.L, len(setup.Trustees))
			// 	}
			// }
			// // now go through all L
			// l := len(setup.Trustees)
			// tid := puid.WithPrefixByte('t')
			// for i := 0; i < setup.Params.L; i++ {
			// 	if i >= l {
			// 		setup.Trustees = append(setup.Trustees, &TrusteeSetup{
			// 			TrusteeID: tid.New(),
			// 			Name:      fmt.Sprintf("Trustee: %d", i+1),
			// 		})
			// 	}
			// 	t := setup.Trustees[i]
			// 	log.Info().Int("index", i).Str("id", t.TrusteeID).Msg("")
			// 	name, err := ui.Ask(fmt.Sprintf(">>> Name for Trustee %d", i+1), &input.Options{
			// 		HideOrder:    true,
			// 		Default:      t.Name,
			// 		ValidateFunc: notEmpty,
			// 		Loop:         true,
			// 	})
			// 	if err != nil {
			// 		log.Fatal().Msgf("Error reading input: %s", err)
			// 	}
			// 	t.Name = name
			// 	// @todo ask for next stage info or skip
			// 	save()
			// }

		},
	}
	interactiveCmd.Flags().StringVar(
		&authorityConfigFile,
		"config",
		"authority-config.json",
		"The file to store authority data",
	)
	interactiveCmd.Flags().Int16Var(&uiPort, "port", 0, "Port for the UI, if 0 will let the OS pick a port")
	interactiveCmd.Flags().BoolVar(&openUI, "open-browser", true, "Automatically attempt to open the system default web browser")
	authorityCmd.AddCommand(interactiveCmd)

	var dataDir string

	var simulateCmd = &cobra.Command{
		Use:   "simulate",
		Short: "Simulate Election Setup Authority",
		Long:  "Simulate the process of creating the election, authority, registrar, trustees and the beginning of a chain.",
		Run: func(cmd *cobra.Command, args []string) {
			// delete the data dir and then rebuild it. making sure we have
			// a clean slate
			os.RemoveAll(dataDir)
			os.MkdirAll(dataDir, 0777)

			numTrustees := 5
			trusteesRequired := 3
			system := &elgamal.ThresholdSystem{
				System: elgamal.DH2048modp224(), //.Astris2048(),
				T:      trusteesRequired - 1,
				L:      numTrustees,
			}

			setup := &astris.PayloadElectionSetup{
				Version:          astris.AstrisProtocolVersion,
				Name:             "Simulated Election",
				Difficulty:       1,
				Params:           system.System,
				TrusteesRequired: trusteesRequired,
				MaxChoices:       1,
				Candidates:       []string{"C1", "C2", "C3", "C4", "C5"},
				Timing: &astris.TimingInfo{
					Timezone: "Europe/London",
					ParameterConfirmation: &astris.TimeBounds{
						Opens:  "2021-04-01T00:00:00", // 1617231600
						Closes: "2021-04-01T23:59:59",
					},
					VoterRegistration: &astris.TimeBounds{
						Opens:  "2021-04-02T00:00:00", // 1617318000
						Closes: "2021-04-02T23:59:59",
					},
					VoteCasting: &astris.TimeBounds{
						Opens:  "2021-04-03T00:00:00", // 1617404400
						Closes: "2021-04-03T23:59:59",
					},
					TallyDecryption: &astris.TimeBounds{
						Opens:  "2021-04-04T00:00:00", // 1617490800
						Closes: "2021-04-04T23:59:59",
					},
				},
				Registrar: createRegistrar(system.System, dataDir),
				Trustees:  make([]*astris.TrusteeSetup, numTrustees),
			}
			// create the trustees.
			trusteePrivate := make([]*TrusteePrivate, len(setup.Trustees))
			// let's keep the Exponents here for ease (Public Info)
			exponents := map[int]crypto.BigIntSlice{}
			for i := range setup.Trustees {
				trusteePrivate[i], setup.Trustees[i] = createTrustee(i+1, system, dataDir)
				exponents[i+1] = setup.Trustees[i].Exponents
			}
			for _, tp := range trusteePrivate {
				// add all the public data to the private bits.
				tp.Threshold.PublicExp = exponents
			}

			var payload bytes.Buffer
			hash, _ := astris.CanonicalJSON.EncodeAndHash(&payload, nil, setup)
			// now turn that into a block to mint.
			genesis := &blockchain.Block{
				Payload: payload.Bytes(),
				Header: &blockchain.BlockHeader{
					PayloadHash:  astris.SliceToID(hash),
					Depth:        0,
					PayloadHint:  uint8(astris.HintElectionSetup),
					EpochSeconds: 1609459200, // "2021-01-01T00:00:00"
				},
			}
			// we have to do the work ourselves for this block
			genesis.Header.Proof, _ = genesis.Header.CalculateProofOfWork(context.TODO(), astris.AstrisWorkLevel)
			genesis.Header.ID = genesis.Header.CalculateBlockID()

			// now create a chain.
			validator := astris.NewElectionValidator(genesis.Header.ID)
			chain, err := blockchain.Create(dataDir, genesis, astris.AstrisWorkLevel, validator)
			if err != nil {
				panic(err)
			}

			log.Info().Str("electionId", chain.ID().String()).Msg("Simulated Election Initialised")
			var param1ts uint32 = 1617231601 // 2021-04-01T00:00:01+01:00
			// now do the parameter confirmation phase part 1
			// for each Trustee encrypt the secret shares for the other trustees.

			for _, ti := range trusteePrivate {
				// each other trustee's shares.
				payload := &astris.PayloadTrusteeShares{
					Index:  ti.Index,
					Shares: make([]*astris.EncryptedShare, 0, len(setup.Trustees)-1),
				}
				for _, tj := range setup.Trustees {
					if ti.Index == tj.Index {
						continue
					}
					sij := ti.Threshold.CreateSecretShare(tj.Index)
					pt := tj.EncKey.Encrypt(sij, nil)
					share := &astris.EncryptedShare{
						Sender:    ti.Index,
						Recipient: tj.Index,
						Point:     pt,
					}
					share.Signature = ti.Keys.Sig.Secret().Sign(share)
					payload.Shares = append(payload.Shares, share)
				}
				// now add the block.
				blk, err := astris.NewBlockBase(astris.HintTrusteeShares, payload)
				if err != nil {
					panic(err)
				}
				if err := chain.Mint(blk, validator.WorkLevel(), param1ts); err != nil {
					panic(err)
				}
				param1ts += 1
			}

			// now do the parameter confirmation phase part 2
			for _, ti := range trusteePrivate {
				// gather the shares and combine into a key.
				// check they are OK first.
				fn := func(j, i int) *big.Int {
					ct := validator.GetEncryptedSecretShare(j, i)
					if ct == nil {
						fmt.Printf("Missing Share For J(%d), I(%d)\n", j, i)
					}
					return ti.Keys.Enc.Secret().Decrypt(ct)
				}
				ti.Threshold.CombineSharesSharedKeys(fn)

				// save the trustee data now.
				writeObjectToFile(dataDir, fmt.Sprintf("simulated-trustee-%d.json", ti.Index), ti)

				// each other trustee's shares.
				payload := &astris.PayloadTrusteePublic{
					Index:      ti.Index,
					ShardKey:   ti.Threshold.ShardKey.Public(),
					ShardProof: ti.Threshold.ShardKey.Secret().ProofOfKnowledge(),
				}
				payload.Signature = ti.Keys.Sig.Secret().Sign(payload)

				// mint the block
				blk, err := astris.NewBlockBase(astris.HintTrusteePublic, payload)
				if err != nil {
					panic(err)
				}
				if err := chain.Mint(blk, validator.WorkLevel(), param1ts); err != nil {
					panic(err)
				}
				param1ts += 1
			}
			// the chain is ready, and we can copy it for as many simluated votes as we want
		},
	}
	simulateCmd.Flags().StringVar(&dataDir, "data-dir", ".", "The path to the directory to store data in")
	authorityCmd.AddCommand(simulateCmd)
}

func createRegistrar(s *elgamal.System, dir string) *astris.RegistrarSetup {
	kp := elgamal.GenerateKeyPair(s)
	r := &astris.RegistrarSetup{
		Name:            "R1",
		RegistrationURL: "https://astris.0x6377.dev/registration",
		SigKey:          kp.Public(),
	}
	r.Signature = kp.Secret().Sign(r)
	// write the keypair to disk so we can load them again later.
	writeObjectToFile(dir, "simulated-registrar.json", kp)
	return r
}

func writeObjectToFile(dir, name string, object interface{}) {
	filename := path.Join(dir, name)
	// ignore error panic later
	f, _ := os.Create(filename)
	defer f.Close()
	if err := astris.CanonicalJSON.Encode(f, object); err != nil {
		panic(err)
	}
}

type TrusteePrivate struct {
	Index     int
	Keys      *elgamal.DerivedKeys
	Threshold *elgamal.PrivateParticipant
}

func createTrustee(index int, s *elgamal.ThresholdSystem, dir string) (*TrusteePrivate, *astris.TrusteeSetup) {
	seed := big.NewInt(int64(index)) // not secure, but easily repeatable
	keys := elgamal.DeriveKeys(s.System, seed)
	coeffs := elgamal.DeriveCoefficients(s.System, seed, s.T)
	exp := elgamal.CreateExponents(s.System, coeffs)
	ts := &astris.TrusteeSetup{
		Index:     index,
		Name:      fmt.Sprintf("T%d", index),
		SigKey:    keys.Sig.Public(),
		EncKey:    keys.Enc.Public(),
		EncProof:  keys.Enc.Secret().ProofOfKnowledge(),
		Exponents: exp,
	}
	ts.Signature = keys.Sig.Secret().Sign(ts)
	private := &TrusteePrivate{
		Index: index,
		Keys:  keys,
		Threshold: &elgamal.PrivateParticipant{
			Sys:    s,
			Index:  index,
			Coeffs: coeffs,
		},
	}
	// we don't need to write to file as we have derived from index.
	// but we will anyway, in case I change that later.
	writeObjectToFile(dir, fmt.Sprintf("simulated-trustee-%d.json", index), private)
	return private, ts
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
