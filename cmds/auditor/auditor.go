package auditor

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/thechriswalker/go-astris/astris"
)

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var electionIdStr string
	// var uiPort int16
	// var openUI bool
	var seedPeers []string
	var listenAddr string
	var peerAddr string
	var dataDir string
	var validateOnly bool

	var cmd = &cobra.Command{
		Use:   "auditor",
		Short: "Election Auditor Node",
		Long:  "Run an Auditor node to validate the election chain and serve it to other nodes",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Starting Election Auditor Process")
			// this should be simple. We should just "start" the p2p node, with the given seeds.
			var electionId astris.ID

			err := electionId.FromString(electionIdStr)
			if err != nil {
				log.Fatal().Err(err).Str("election", electionIdStr).Msg("given election id was not valid")
			}

			node, err := astris.Node(
				electionId,
				astris.WithListenAddr(listenAddr),
				astris.WithSeedPeers(seedPeers),
				astris.WithExternalAddr(peerAddr),
				astris.WithDataDir(dataDir),
				astris.WithValidateOnly(validateOnly),
			)

			if err != nil {
				log.Fatal().Err(err).Msg("Failed to create Auditor Node")
			}

			// should probably have this one cancel on sigint
			ctx := context.Background()

			if err := node.Run(ctx); err != nil {
				log.Fatal().Err(err).Msg("Failed to create Auditor Node")
			}
			result := node.GetResult()
			// output to stdout as JSON
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(result)
			enc.Encode(node.GetBenchmarks())
			enc.Encode(node.GetTimings())
		},
	}

	// for now all configuration is done via command line arguments, rather than config files.
	// dotenv and another spf13 lib (viper?) could help here.
	cmd.Flags().StringVar(&dataDir, "data-dir", ".", "The path to the directory to store data in")
	cmd.Flags().StringVar(&electionIdStr, "election-id", "", "The Election ID (as base64url)")
	cmd.Flags().StringVar(&peerAddr, "public-peer-addr", "", "The address other nodes will use to connect (will default to the local listen addr)")
	cmd.Flags().StringVar(&listenAddr, "local-peer-addr", "localhost:0", "The address this node will bind to listen on (note that port :0 will let the OS choose an emphemeral port)")
	cmd.Flags().StringSliceVar(&seedPeers, "seeds", []string{}, "Seed peers to connect to initially")
	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "If true, exit as soon as a valid or invalid chain is determined")

	// not sure about the UI yet...
	// cmd.Flags().Int16Var(&uiPort, "port", 0, "Port for the UI, if 0 will let the OS pick a port")
	// cmd.Flags().BoolVar(&openUI, "open-browser", true, "Automatically attempt to open the system default web browser")
	rootCmd.AddCommand(cmd)
}
