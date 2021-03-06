package main

import (
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/thechriswalker/go-astris/cmds/authority"
)

// These variables will be linked in at build time
var (
	BuildDate string
	Commit    string
	Version   string
)

func preamble(cmd *cobra.Command, args []string) {
	// preamble dump some info
	log.Debug().
		Str("version", Version).
		Str("license", "GPLv3+").
		Msg("Astris Voting")

	log.Debug().
		Str("commit", Commit[0:8]).
		Str("built", BuildDate).
		Str("arch", runtime.GOARCH).
		Str("os", runtime.GOOS).
		Msg("Build Info")
}

const timeFormatMs = "2006-01-02T15:04:05.000Z07:00"
const timeFormatLocal = "2006-01-02 15:04:05.000"

func main() {
	// configure the logger.
	// remember pretty logs are only good on the console
	zerolog.TimeFieldFormat = timeFormatMs
	log.Logger = log.Output(zerolog.NewConsoleWriter(func(cw *zerolog.ConsoleWriter) {
		cw.TimeFormat = timeFormatLocal
	}))

	// initialise the cobra framework for the command.
	var rootCmd = &cobra.Command{
		Use:              "astris",
		Short:            "Astris P2P Voting",
		Version:          Version,
		PersistentPreRun: preamble,
	}

	authority.Register(rootCmd)
	// commands:
	//
	// - authority: setup an election, accept trustee and registrar data create genesis block
	// - trustee: help setup the election, create key shards, setup phases, final tally partial decryption
	// - registrar: help setup the election, run authentication server for voters, voter lists
	// - voter: register to vote, create keys, create votes
	//
	// - node: run the chain p2p node, accepting blocks and validating them. This process
	//			also becomes an "auditor" role, as once the whole chain is present (or downloaded separately)
	//       	this process will validate the chain, producing a result (if there is enough chain)
	//
	// - auditor: like the node (probably a sub-command, or vice-versa), but only attempts to download the full chain and validate it and give
	//			progress (like, how far through the election we are) or show errors.

	// This is the sub command to run the p2p node - e.g. an auditor.
	// 	var nodeCmd = &cobra.Command{
	// 		Use:   "node",
	// 		Short: "Run the P2P Node and Web Service",
	// 		Long: `Start a P2P Node, for each election we care about
	// attempt to get the longest chain and then continue to participate
	// in the network, accepting and validating blocks.
	// `,
	// 		Args: cobra.NoArgs,
	// 		Run: func(cmd *cobra.Command, args []string) {
	// 			preamble()
	// 			log.Debug().
	// 				Str("data-dir", dataDir).
	// 				Msg("Initialising Astris P2P Node")

	// 			// load DB, find any chains and validate them
	// 			log.Info().
	// 				Str("blockchain", dataDir+"/blockchain.db").
	// 				Msg("Verifying local blockchain")

	// 			if peerAddr == "" {
	// 				peerAddr = grpcAddr
	// 			}
	// 			log.Info().
	// 				Str("addr", grpcAddr).
	// 				Str("public_addr", peerAddr).
	// 				Msg("Starting local GRPC service")

	// 			log.Info().
	// 				Str("addr", webAddr).
	// 				Msg("Starting local Web interface")

	// 			log.Info().Msg("Connecting to seed peers")

	// 			log.Info().Msg("Initialisation complete")

	// 			// main loop
	// 			log.Fatal().Msg("Not Implement Yet")

	// 			log.Info().Msg("Astris node shutting down")
	// 		},
	// 	}

	// 	var registrarCmd = &cobra.Command{
	// 		Use:   "registrar",
	// 		Short: "Run a simple eligibility service",
	// 		Long: `Astris requires an external authority to mandate voter eligibility.

	// This command will run a very simple authority server which we can use to
	// confirm eligibility.

	// It is not designed to be rigorous or secure but to provide an interface so we
	// can actually use the P2P service.
	// `,
	// 		Run: func(cmd *cobra.Command, args []string) {
	// 			preamble()
	// 			log.Debug().
	// 				Str("data-dir", dataDir).
	// 				Msg("Initialising Astris Eligibility Service")
	// 			log.Fatal().Msg("Not Implement Yet")
	// 		},
	// 	}

	// 	var verifyCmd = &cobra.Command{
	// 		Use:   "verify",
	// 		Short: "Verify the result of an election",
	// 		Long: `Verify the election result in one of three ways:

	//  - Against our local copy of the blockchain for this election.
	//    Use the data we have to validate the blockchain is valid, contains
	//    our election and has been tallied correctly.

	//  - Against a provided blockchain for this election.
	//    Use a user-provided blockchain file and validate that it contains the
	//    election data and has been tallied correctly.

	//    - Against a peer-sourced blockchain.
	//    Connect to the P2P network and attempt to download the blockchain for
	//    the election in question, validate the chain and ensure the tally is
	//    correct.
	// `,
	// 		Run: func(cmd *cobra.Command, args []string) {
	// 			preamble()
	// 			log.Debug().
	// 				Str("data-dir", dataDir).
	// 				Msg("Validating Election Result")
	// 			log.Fatal().Msg("Not Implement Yet")
	// 		},
	// 	}

	// the data-dir is a persistent flag, it applies to all subcommands
	//	rootCmd.PersistentFlags().StringVar(&dataDir, "data", "$HOME/.astris/data", "The path to the directory to store all data")

	// nodeCmd.Flags().StringVar(&grpcAddr, "grpc-addr", "0:8081", "The address to bind the GRPC service to")
	// nodeCmd.Flags().StringVar(&peerAddr, "peer-addr", "", "The publically accessible peer address for inbound GRPC connections, defaults to the grpc address")

	// // make sure our commands are available.
	// rootCmd.AddCommand(nodeCmd, authorityCmd, verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Err(err).Msg("An Error Occured")
		os.Exit(1)
	}
}
