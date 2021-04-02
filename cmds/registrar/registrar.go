package registrar

import (
	"github.com/spf13/cobra"
)

// Register the election setup command
func Register(rootCmd *cobra.Command) {
	var regCmd = &cobra.Command{
		Use:   "registrar",
		Short: "Registrar Commands",
	}

	rootCmd.AddCommand(regCmd)
}
