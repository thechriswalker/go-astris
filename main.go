package main

import (
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/thechriswalker/go-astris/astris"
	"github.com/thechriswalker/go-astris/cmds/auditor"
	"github.com/thechriswalker/go-astris/cmds/authority"
	"github.com/thechriswalker/go-astris/cmds/registrar"
	"github.com/thechriswalker/go-astris/cmds/trustee"
	"github.com/thechriswalker/go-astris/cmds/voter"
)

func preamble(cmd *cobra.Command, args []string) {
	// preamble dump some info
	log.Info().
		Str("version", astris.Version).
		Str("protocol", astris.AstrisProtocolVersion).
		Str("license", "GPLv3+").
		Msg("Astris Voting")

	log.Debug().
		Str("commit", astris.Commit[0:8]).
		Str("built", astris.BuildDate).
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
		cw.NoColor = true
	}))

	// initialise the cobra framework for the command.
	var rootCmd = &cobra.Command{
		Use:              "astris",
		Short:            "Astris P2P Voting",
		Version:          astris.Version,
		PersistentPreRun: preamble,
	}

	if os.Getenv("DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

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

	auditor.Register(rootCmd)
	authority.Register(rootCmd)
	registrar.Register(rootCmd)
	trustee.Register(rootCmd)
	voter.Register(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Err(err).Msg("An Error Occured")
		os.Exit(1)
	}
}
