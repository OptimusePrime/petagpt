package index

import (
	"errors"
	"fmt"
	"log"

	"github.com/OptimusePrime/petagpt/cmd/index"
	bleve "github.com/blevesearch/bleve/v2"
)

type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

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

func AddDocumentsToBleveIndex(indexPath string, docs ...Document) error {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(index.Close())
	}()

	batch := index.NewBatch()
	for _, doc := range docs {
		err = batch.Index(doc.ID, doc)
		if err != nil {
			return err
		}
	}
	if err = index.Batch(batch); err != nil {
		return err
	}

	return nil
}

func SearchBleveIndex(indexPath string, queryString string, field string, size int) (*bleve.SearchResult, error) {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(index.Close())
	}()

	query := bleve.NewMatchQuery("Bleve provides full-text search capabilities.")
	query.SetField("content")
	searchRequest := bleve.NewSearchRequestOptions(query, 5, 0, false)
	searchRequest.Fields = []string{"title", "content"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	return searchResult, nil
}

func TestBleve() {
	mapping := bleve.NewIndexMapping()
	mapping.ScoringModel = "bm25"
	mapping.TypeField = "type"
	mapping.DefaultAnalyzer = "en"
	index, err := bleve.New("bm25.bleve", mapping)
	if err != nil {
		log.Fatal(err)
	}
	defer index.Close()

	documents := []Document{
		{
			ID:      "doc1",
			Title:   "Bleve documentation",
			Content: "Bleve provides full-text search capabilities.",
		},
		{
			ID:      "doc2",
			Title:   "Bleve documentation 2",
			Content: "Bleve 2 provides full-text search capabilities.",
		},
		{
			ID:      "doc3",
			Title:   "Elasticsearch documentation",
			Content: "Elasticsearch provides full-text search capabilities as well.",
		},
	}

	batch := index.NewBatch()
	for _, doc := range documents {
		batch.Index(doc.ID, doc)
	}
	if err := index.Batch(batch); err != nil {
		log.Fatal(err)
	}

}
