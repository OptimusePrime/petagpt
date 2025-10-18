package index

import (
	"context"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
)

func CreateIndex(ctx context.Context, params sqlc.CreateIndexParams) error {
	queries := sqlc.New(db.MainDB)

	_, err := queries.CreateIndex(ctx, params)
	if err != nil {
		return err
	}

	return nil
}
