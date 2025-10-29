package document

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/spf13/cobra"
)

var numWorkers int
var chunkSize int

var documentAddCommand = &cobra.Command{
	Use:   "add",
	Short: "Add new document(s) to a document index",
	RunE: func(cmd *cobra.Command, args []string) error {
		dc, err := parser.NewDocumentChunker(cmd.Context(), numWorkers)
		if err != nil {
			return fmt.Errorf("failed to create a document chunker: %w", err)
		}

		indexName := args[0]

		queries := sqlc.New(db.MainDB)

		idx, err := queries.GetIndexByName(cmd.Context(), indexName)
		if err != nil {
			return fmt.Errorf("failed to find idx: %w", err)
		}

		for _, docPath := range args[1:] {
			doc, err := os.Open(docPath)
			if err != nil {
				return fmt.Errorf("failed to open document: %s: %w", docPath, err)
			}

			docData, err := io.ReadAll(doc)
			if err != nil {
				return fmt.Errorf("failed to read document: %s: %w", docPath, err)
			}

			docStat, err := doc.Stat()
			if err != nil {
				return fmt.Errorf("failed getting file info: %s: %w", docPath, err)
			}

			chunks, err := parser.ProcessDocument(cmd.Context(), docData, filepath.Ext(docPath), dc, chunkSize)
			if err != nil {
				return fmt.Errorf("failed chunking document: %s: %w", docPath, err)
			}

			document, err := queries.CreateDocument(cmd.Context(), sqlc.CreateDocumentParams{
				IndexID:  idx.ID,
				Filepath: docPath,
				Filetype: filepath.Ext(docPath),
				Filesize: docStat.Size(),
			})
			if err != nil {
				return fmt.Errorf("failed creating document in database: %s: %w", docPath, err)
			}

			for _, c := range chunks {
				_, err = queries.CreateChunk(cmd.Context(), sqlc.CreateChunkParams{
					DocumentID: document.ID,
					Content: sql.NullString{
						String: c.Content,
					},
					Context: sql.NullString{
						String: c.Context,
					},
				})
				if err != nil {
					return fmt.Errorf("failed creating chunk in database: %s: %w", docPath, err)
				}
			}

			err = index.AddChunksToBleveIndex(idx.Path, chunks...)
			if err != nil {
				return fmt.Errorf("failed adding chunks to BM25 index: %s: %w", idx.Path, err)
			}

			err = index.AddChunksToChromaCollection(cmd.Context(), indexName, chunks...)
			if err != nil {
				return fmt.Errorf("failed adding chunks to Chroma collection: %s: %w", indexName, err)
			}
		}

		//dc, err := parser.NewDocumentChunker(cmd.Context(), numWorkers)
		//if err != nil {
		//	return fmt.Errorf("failed to create a document chunker: %w", err)
		//}
		//defer func() {
		//	err = errors.Join(err, dc.Shutdown())
		//}()
		//
		//wg := sync.WaitGroup{}
		//wg.Add(len(args))
		//
		//for _, docPath := range args {
		//	go func() {
		//		docContent, err := os.ReadFile(docPath)
		//		if err != nil {
		//			fmt.Println(err)
		//		}
		//
		//		_, err = parser.parseDocument(cmd.Context(), docContent, dc)
		//		if err != nil {
		//			fmt.Println(err)
		//		}
		//
		//		wg.Done()
		//	}()
		//}
		//
		//wg.Wait()

		return nil
	},
}

func newDocumentAddCmd() *cobra.Command {
	documentAddCommand.Flags().IntVarP(&numWorkers, "num_workers", "w", 4, "Specify the number of workers for sentence segmentation")
	documentAddCommand.Flags().IntVarP(&chunkSize, "chunk_size", "c", 50, "Size of the chunks in number of sentences")

	return documentAddCommand
}
