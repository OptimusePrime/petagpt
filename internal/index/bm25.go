package index

import (
	"fmt"
	"log"

	bleve "github.com/blevesearch/bleve/v2"
)

type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
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

	query := bleve.NewMatchQuery("Bleve provides full-text search capabilities.")
	query.SetField("content")
	searchRequest := bleve.NewSearchRequestOptions(query, 5, 0, false)
	//searchRequest.
	//searchRequest.From = 5
	searchRequest.Fields = []string{"title", "content"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(searchResult.Hits[0].Score)
}
