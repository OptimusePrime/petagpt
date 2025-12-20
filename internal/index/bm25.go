package index

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/blevesearch/bleve/v2"
)

func CreateBleveIndex(indexPath string, defaultAnalyzer string) (bleve.Index, error) {
	mapping := bleve.NewIndexMapping()
	mapping.ScoringModel = "bm25"
	mapping.TypeField = "type"
	mapping.DefaultAnalyzer = defaultAnalyzer

	index, err := bleve.New(indexPath, mapping)
	if err != nil {
		return nil, err
	}

	err = index.Close()
	if err != nil {
		return nil, err
	}

	return index, nil
}

func DeleteBleveIndex(indexPath string) error {
	err := os.RemoveAll(indexPath)
	if err != nil {
		return fmt.Errorf("failed to delete Bleve index: %w", err)
	}

	return nil
}

func AddChunksToBleveIndex(indexPath string, docs ...parser.Chunk) error {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open Bleve index: %w", err)
	}

	defer func() {
		err = errors.Join(err, index.Close())
	}()

	batch := index.NewBatch()
	for _, doc := range docs {
		err = batch.Index(doc.SHA256(), doc)
		if err != nil {
			return fmt.Errorf("failed to add chunk to Bleve index: %w", err)
		}
	}
	if err = index.Batch(batch); err != nil {
		return fmt.Errorf("failed to add chunks to Bleve index: %w", err)
	}

	return nil
}

func RemoveChunksFromBleveIndex(ctx context.Context, indexPath string, chunksIDs []string) error {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open Bleve index: %w", err)
	}

	defer func() {
		err = errors.Join(err, index.Close())
	}()

	batch := index.NewBatch()
	for _, docID := range chunksIDs {
		batch.Delete(docID)
	}
	if err = index.Batch(batch); err != nil {
		return fmt.Errorf("failed to remove chunks from Bleve index: %w", err)
	}

	return nil
}

func SearchBleveIndex(indexPath string, queryString string, topN int) (*bleve.SearchResult, error) {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = errors.Join(err, index.Close())
	}()

	query := bleve.NewMatchQuery(queryString)

	searchRequest := bleve.NewSearchRequestOptions(query, max(topN, 100), 0, true)
	searchRequest.Fields = []string{"*"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	return searchResult, nil
}
