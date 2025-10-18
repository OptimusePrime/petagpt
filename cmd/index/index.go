package index

import "github.com/spf13/cobra"

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Allows you to inspect and manipulate the chatbot's index of documents (knowledgebase)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func NewCommand() *cobra.Command {
	indexCmd.AddCommand(newIndexAddCommand())

	return indexCmd
}
