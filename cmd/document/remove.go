package document

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	chroma "github.com/OptimusePrime/chroma-go/pkg/api/v2"
	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/spf13/cobra"
)

func newDocumentRemoveCmd() *cobra.Command {
	var (
		idxName string
	)

	documentRemoveCommand := &cobra.Command{
		Use:   "remove",
		Short: "Remove a document from the specified index",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("you must provide at least one document path")
			}

			queries := sqlc.New(db.MainDB)

			var chunkIDs []string
			var dbDoc sqlc.Document

			for _, docPath := range args {
				doc, err := os.Open(docPath)
				if err != nil {
					return fmt.Errorf("failed to open document: %s: %w", docPath, err)
				}

				defer doc.Close()

				docData, err := io.ReadAll(doc)
				if err != nil {
					return fmt.Errorf("failed to read document: %s: %w", docPath, err)
				}

				checksum := sha256.Sum256(docData)

				dbDoc, err := queries.GetDocumentBySHA256(cmd.Context(), base64.StdEncoding.EncodeToString(checksum[:]))
				if err != nil {
					return fmt.Errorf("failed to find document: %w", err)
				}

				chunks, err := queries.GetChunksByDocumentID(cmd.Context(), dbDoc.ID)
				if err != nil {
					return fmt.Errorf("failed to find chunks: %w", err)
				}

				for _, chunk := range chunks {
					chunkIDs = append(chunkIDs, chunk.IndexingID.String)
				}

			}

			// Convert []string to []chroma.DocumentID
			docIDs := make([]chroma.DocumentID, len(chunkIDs))
			for i, id := range chunkIDs {
				docIDs[i] = chroma.DocumentID(id)
			}

			err := index.RemoveChunksFromChromaCollection(cmd.Context(), idxName, docIDs)
			if err != nil {
				return fmt.Errorf("failed deleting chunks from chroma collection: %w", err)
			}
			err = index.RemoveChunksFromBleveIndex(cmd.Context(), idxName, chunkIDs)
			if err != nil {
				return fmt.Errorf("failed deleting chunks from bleve index: %w", err)
			}

			if err := queries.DeleteDocument(cmd.Context(), dbDoc.ID); err != nil {
				return fmt.Errorf("failed to delete document: %w", err)
			}

			return nil
		},
	}

	documentRemoveCommand.Flags().StringVarP(&idxName, "index", "i", "", "The name of the index to remove the document from")
	return documentRemoveCommand
}
