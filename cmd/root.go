package cmd

import (
	"context"
	"os"

	"github.com/OptimusePrime/petagpt/cmd/document"
	"github.com/OptimusePrime/petagpt/cmd/index"
	"github.com/OptimusePrime/petagpt/cmd/serve"
	"github.com/OptimusePrime/petagpt/configs"
	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "petagpt",
		Short: "PetaGPT is an enterprise knowledgebase AI assistant",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := configs.InitConfig(cmd)
			if err != nil {
				return err
			}

			err = db.InitDatabase(context.Background())
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configs.CfgFile, "config", "", "config file (default is $HOME/.petagpt.yaml)")

	rootCmd.AddCommand(serve.NewCommand())
	rootCmd.AddCommand(index.NewCommand())
	rootCmd.AddCommand(document.NewCommand())
}
