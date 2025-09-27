package index

import "github.com/spf13/cobra"

var indexAddCommand = &cobra.Command{
	Use:   "add",
	Short: "Add new document(s) to a document index",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
