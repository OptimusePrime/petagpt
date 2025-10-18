package index

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var name string
var description string

var indexAddCommand = &cobra.Command{
	Use:   "add",
	Short: "Create a new document index",
	RunE: func(cmd *cobra.Command, args []string) error {

		if len(strings.TrimSpace(name)) == 0 {
			return fmt.Errorf("you must provide a name for the index")
		}

		err := index.CreateIndex(cmd.Context(), sqlc.CreateIndexParams{
			Name: name,
			Description: sql.NullString{
				String: description,
				Valid:  true,
			},
		})
		if err != nil {
			if errors.Is(err, sqlite3.ErrConstraintUnique) {
				return fmt.Errorf("index names must be unique: %w", err)
			}

			return fmt.Errorf("failed to create new index: %w", err)
		}

		return nil
	},
}

func newIndexAddCommand() *cobra.Command {
	indexAddCommand.Flags().StringVarP(&name, "name", "n", "", "The name of the index, must be unique")
	indexAddCommand.Flags().StringVarP(&description, "description", "d", "", "The description of the index")

	return indexAddCommand
}
