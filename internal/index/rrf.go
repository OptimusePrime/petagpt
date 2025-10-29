package index

import (
	"context"
	"path/filepath"
	"slices"

	"github.com/spf13/viper"
)

const RRF_K = 60

func SearchIndex(ctx context.Context, indexName string, query string, topN int) (*SearchResult, error) {
	chromaResult, err := SearchChromaCollection(ctx, indexName, topN, query)
	if err != nil {
		return nil, err
	}

	blevePath := filepath.Join(viper.GetString("data_dir"), "indexes", indexName+".bleve")
	bm25Result, err := SearchBleveIndex(blevePath, query, topN)
	if err != nil {
		return nil, err
	}

	chromaGroup := chromaResult.GetDocumentsGroups()[0]

	chromaSearchResult := new(SearchResult)
	bm25SearchResult := new(SearchResult)

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
				ID:      hit.ID,
				Title:   hit.Fields["title"].(string),
				Content: hit.Fields["content"].(string),
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
