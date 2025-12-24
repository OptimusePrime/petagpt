package document

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newDocumentAddCmd() *cobra.Command {
	var (
		numWorkers   int
		chunkSize    int
		idxName      string
		requestDelay int
	)

	documentAddCommand := &cobra.Command{
		Use:   "add",
		Short: "Add new document(s) to a document index",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("you must provide at least one document path")
			}

			dc, err := parser.NewDocumentChunker(cmd.Context(), numWorkers, viper.GetInt("context_llm.max_concurrent_requests"))
			if err != nil {
				return fmt.Errorf("failed to create a document chunker: %w", err)
			}

			queries := sqlc.New(db.MainDB)

			idx, err := queries.GetIndexByName(cmd.Context(), idxName)
			if err != nil {
				return fmt.Errorf("failed to find idx: %w", err)
			}

			for _, docPath := range args {
				doc, err := os.Open(docPath)
				if err != nil {
					return fmt.Errorf("failed to open document: %s: %w", docPath, err)
				}

				docData, err := io.ReadAll(doc)
				if err != nil {
					return fmt.Errorf("failed to read document: %s: %w", docPath, err)
				}

				checksum := sha256.Sum256(docData)

				docStat, err := doc.Stat()
				if err != nil {
					return fmt.Errorf("failed getting file info: %s: %w", docPath, err)
				}

				chunks, err := parser.ProcessDocument(cmd.Context(), docData, filepath.Base(docPath), dc, chunkSize, requestDelay)
				if err != nil {
					return fmt.Errorf("failed chunking document: %s: %w", docPath, err)
				}

				err = index.AddChunksToBleveIndex(idx.Path, chunks...)
				if err != nil {
					return fmt.Errorf("failed adding chunks to BM25 index: %s: %w", idx.Path, err)
				}

				err = index.AddChunksToChromaCollection(cmd.Context(), idxName, chunks...)
				if err != nil {
					return fmt.Errorf("failed adding chunks to Chroma collection: %s: %w", idxName, err)
				}

				document, err := queries.CreateDocument(cmd.Context(), sqlc.CreateDocumentParams{
					IndexID:    idx.ID,
					Filepath:   docPath,
					Filetype:   filepath.Ext(docPath),
					Filesize:   docStat.Size(),
					Filesha256: base64.StdEncoding.EncodeToString(checksum[:]),
				})
				if err != nil {
					return fmt.Errorf("failed creating document in database: %s: %w", docPath, err)
				}

				for _, c := range chunks {
					_, err = queries.CreateChunk(cmd.Context(), sqlc.CreateChunkParams{
						DocumentID: document.ID,
						Content:    c.Content,
						Context:    c.Context,
						IndexingID: c.ID,
					})
					if err != nil {
						return fmt.Errorf("failed creating chunk in database: %s: %w", docPath, err)
					}
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

	documentAddCommand.Flags().IntVarP(&numWorkers, "num_workers", "w", 8, "Specify the number of workers for sentence segmentation")
	documentAddCommand.Flags().IntVarP(&chunkSize, "chunk_size", "c", 50, "Size of the chunks in number of sentences")
	documentAddCommand.Flags().StringVarP(&idxName, "index", "i", "", "The name of the index to add the document to")
	documentAddCommand.Flags().IntVarP(&requestDelay, "request_delay", "d", 0, "Delay between requests to the LLM service in milliseconds")

	return documentAddCommand
}
