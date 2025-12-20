package index

import (
	"fmt"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/spf13/cobra"
)

func newIndexRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove an index",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("you must provide at least one index name")
			}

			queries := sqlc.New(db.MainDB)

			for _, idxName := range args {
				idx, err := queries.GetIndexByName(cmd.Context(), idxName)
				if err != nil {
					return fmt.Errorf("failed to find index: %w", err)
				}

				err = index.DeleteBleveIndex(idx.Path)
				if err != nil {
					return fmt.Errorf("failed to delete Bleve index: %w", err)
				}
				err = index.DeleteChromaCollection(cmd.Context(), idx.Name)
				if err != nil {
					return fmt.Errorf("failed to delete Chroma collection: %w", err)
				}
				err = queries.DeleteIndex(cmd.Context(), idx.ID)
				if err != nil {
					return fmt.Errorf("failed to delete index from database: %w", err)
				}
			}

			return nil
		},
	}
}
