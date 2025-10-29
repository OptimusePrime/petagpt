package index

import (
	"errors"
	"log"

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

func AddChunksToBleveIndex(indexPath string, docs ...parser.Chunk) error {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, index.Close())
	}()

	batch := index.NewBatch()
	for _, doc := range docs {
		err = batch.Index(doc.SHA256(), doc)
		if err != nil {
			return err
		}
	}
	if err = index.Batch(batch); err != nil {
		return err
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
	query.SetField("content")
	searchRequest := bleve.NewSearchRequestOptions(query, max(topN, 100), 0, false)
	searchRequest.Fields = []string{"title", "content"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	return searchResult, nil
}
