package index

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
)

type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (d Document) SHA256() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(d.Content)))
}

type SearchResult struct {
	Documents []SearchDocument
}

type SearchDocument struct {
	Rank int `json:"rank"`
	Document
}

func CreateIndex(ctx context.Context, params sqlc.CreateIndexParams) error {
	queries := sqlc.New(db.MainDB)

	_, err := queries.CreateIndex(ctx, params)
	if err != nil {
		return err
	}

	return nil
}
