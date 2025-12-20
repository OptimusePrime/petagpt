package document

import "github.com/spf13/cobra"

var documentCmd = &cobra.Command{
	Use:   "document",
	Short: "Allows you control which documents are added to indexes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func NewCommand() *cobra.Command {
	documentCmd.AddCommand(newDocumentAddCmd())
	documentCmd.AddCommand(newDocumentRemoveCmd())

	return documentCmd
}
