package index

import (
	"context"
	"path/filepath"
	"slices"

	"github.com/spf13/viper"
)

const RRF_K = 60

func SearchIndex(ctx context.Context, indexName string, query string, topN int) (*SearchResult, error) {
	//fmt.Println("Hello")
	chromaResult, err := SearchChromaCollection(ctx, indexName, topN, query)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Hello 2")

	blevePath := filepath.Join(viper.GetString("data_dir"), "bm25", indexName+".bleve")
	//fmt.Println(blevePath)
	bm25Result, err := SearchBleveIndex(blevePath, query, topN)
	if err != nil {
		return nil, err
	}

	chromaGroup := chromaResult.GetDocumentsGroups()[0]
	//fmt.Println(chromaGroup[0].ContentString())

	chromaSearchResult := new(SearchResult)
	bm25SearchResult := new(SearchResult)
	//fmt.Println(bm25Result.Hits[0].Fields["Content"])

	for i, doc := range chromaGroup {
		chromaSearchResult.Documents = append(chromaSearchResult.Documents, SearchDocument{
			Rank: i + 1,
			Document: Document{
				Content: doc.ContentString(),
			},
		})
	}

	for i, hit := range bm25Result.Hits {
		bm25SearchResult.Documents = append(bm25SearchResult.Documents, SearchDocument{
			Rank: i + 1,
			Document: Document{
				ID: hit.ID,
				//Title:   hit.Fields["title"].(string),
				Content: hit.Fields["Content"].(string),
			},
		})
	}

	finalResult := rrf(chromaSearchResult, bm25SearchResult)

	return finalResult, nil
}

func rrf(chromaResult *SearchResult, bm25Result *SearchResult) *SearchResult {
	var finalResult []SearchDocument
	finalScores := make(map[string]float64)

	for i, doc := range chromaResult.Documents {
		finalScores[doc.SHA256()] += 1.0 / (RRF_K + float64(i+1))

		if !slices.ContainsFunc(finalResult, func(s SearchDocument) bool { return s.SHA256() == doc.SHA256() }) {
			finalResult = append(finalResult, doc)
		}
	}

	for i, doc := range bm25Result.Documents {
		finalScores[doc.SHA256()] += 1.0 / (RRF_K + float64(i+1))

		if !slices.ContainsFunc(finalResult, func(s SearchDocument) bool { return s.SHA256() == doc.SHA256() }) {
			finalResult = append(finalResult, doc)
		}
	}

	slices.SortFunc(finalResult, func(a, b SearchDocument) int {
		if finalScores[a.SHA256()] > finalScores[b.SHA256()] {
			return -1
		} else if finalScores[a.SHA256()] < finalScores[b.SHA256()] {
			return 1
		} else {
			return 0
		}
	})

	return &SearchResult{Documents: finalResult}
}
