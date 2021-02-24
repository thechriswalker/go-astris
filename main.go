package main

import (
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// These variables will be linked in at build time
var (
	BuildDate string
	Commit    string
	Version   string
)

func preamble() {
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
		Use:     "astris",
		Short:   "Astris P2P Voting",
		Version: Version,
	}

	// same for the data dir
	var dataDir string

	var grpcAddr string
	var webAddr string
	var peerAddr string

	// This is the sub command to run the p2p node
	var nodeCmd = &cobra.Command{
		Use:   "server",
		Short: "Run the P2P Node and Web Service",
		Long: `Start a P2P Node, for each election we care about
attempt to get the longest chain and then continue to participate
in the network, accepting and validating blocks.

This mode also starts a web-interface to allow:

- new election creation
- voting for eligible voters
- status reporting of the server and the election progress
- combining results into a final tally and the iterative partial tally decryption
`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			preamble()
			log.Debug().
				Str("data-dir", dataDir).
				Msg("Initialising Astris P2P Node")

			// load DB, find any chains and validate them
			log.Info().
				Str("blockchain", dataDir+"/blockchain.db").
				Msg("Verifying local blockchain")

			if peerAddr == "" {
				peerAddr = grpcAddr
			}
			log.Info().
				Str("addr", grpcAddr).
				Str("public_addr", peerAddr).
				Msg("Starting local GRPC service")

			log.Info().
				Str("addr", webAddr).
				Msg("Starting local Web interface")

			log.Info().Msg("Connecting to seed peers")

			log.Info().Msg("Initialisation complete")

			// main loop
			log.Fatal().Msg("Not Implement Yet")

			log.Info().Msg("Astris node shutting down")
		},
	}

	var authorityCmd = &cobra.Command{
		Use:   "authority",
		Short: "Run a simple eligibility service",
		Long: `Astris requires an external authority to mandate voter eligibility.

This command will run a very simple authority server which we can use to
confirm eligibility.

It is not designed to be rigorous or secure but to provide an interface so we
can actually use the P2P service.
`,
		Run: func(cmd *cobra.Command, args []string) {
			preamble()
			log.Debug().
				Str("data-dir", dataDir).
				Msg("Initialising Astris Eligibility Service")
			log.Fatal().Msg("Not Implement Yet")
		},
	}

	var verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Verify the result of an election",
		Long: `Verify the election result in one of three ways:

 - Against our local copy of the blockchain for this election.
   Use the data we have to validate the blockchain is valid, contains
   our election and has been tallied correctly.

 - Against a provided blockchain for this election.
   Use a user-provided blockchain file and validate that it contains the
   election data and has been tallied correctly.

   - Against a peer-sourced blockchain.
   Connect to the P2P network and attempt to download the blockchain for
   the election in question, validate the chain and ensure the tally is
   correct.
`,
		Run: func(cmd *cobra.Command, args []string) {
			preamble()
			log.Debug().
				Str("data-dir", dataDir).
				Msg("Validating Election Result")
			log.Fatal().Msg("Not Implement Yet")
		},
	}

	// the data-dir is a persistent flag, it applies to all subcommands
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", "$HOME/.astris/data", "The path to the directory to store all data")

	nodeCmd.Flags().StringVar(&grpcAddr, "grpc-addr", "0:8081", "The address to bind the GRPC service to")
	nodeCmd.Flags().StringVar(&webAddr, "web-addr", "0:8080", "The address to bind the web interface to")
	nodeCmd.Flags().StringVar(&peerAddr, "peer-addr", "", "The publically accessible peer address for inbound GRPC connections, defaults to the grpc address")

	// make sure our commands are available.
	rootCmd.AddCommand(nodeCmd, authorityCmd, verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Err(err).Msg("An Error Occured")
		os.Exit(1)
	}
}
