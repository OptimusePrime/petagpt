package document

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/spf13/cobra"
)

var numWorkers int

var documentAddCommand = &cobra.Command{
	Use:   "add",
	Short: "Add new document(s) to a document index",
	RunE: func(cmd *cobra.Command, args []string) error {
		dc, err := parser.NewDocumentChunker(cmd.Context(), numWorkers)
		if err != nil {
			return fmt.Errorf("failed to create a document chunker: %w", err)
		}
		defer func() {
			err = errors.Join(err, dc.Shutdown())
		}()

		wg := sync.WaitGroup{}
		wg.Add(len(args))

		for _, docPath := range args {
			go func() {
				docContent, err := os.ReadFile(docPath)
				if err != nil {
					fmt.Println(err)
				}

				err = parser.ParseDocument(cmd.Context(), docContent, dc)
				if err != nil {
					fmt.Println(err)
				}

				wg.Done()
			}()
		}

		wg.Wait()

		return nil
	},
}

func newDocumentAddCmd() *cobra.Command {
	documentAddCommand.Flags().IntVarP(&numWorkers, "num_workers", "w", 4, "Specify the number of workers for sentence segmentation")

	return documentAddCommand
}
