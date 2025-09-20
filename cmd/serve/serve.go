package serve

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the PetaGPT server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := viper.GetInt("port")

		fmt.Printf("Starting PetaGPT server on port %d...\n", port)

		return nil
	},
}

func NewCommand() *cobra.Command {
	return serveCmd
}
