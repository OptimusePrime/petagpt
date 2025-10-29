package index

import (
	"context"
	"errors"

	chroma "github.com/OptimusePrime/chroma-go/pkg/api/v2"
	"github.com/OptimusePrime/chroma-go/pkg/embeddings/vllm"
	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/spf13/viper"
)

func CreateChromaCollection(ctx context.Context, name string) error {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	vllmEf, err := vllm.NewVLLMEmbeddingFunctionFromOptions(
		vllm.WithModel(viper.GetString("embedding_service.model")),
		vllm.WithBaseURL(viper.GetString("embedding_service.base_url")),
		vllm.WithAPIKey(viper.GetString("embedding_service.api_key")),
	)
	if err != nil {
		return err
	}

	_, err = client.CreateCollection(ctx, name, chroma.WithEmbeddingFunctionCreate(vllmEf))
	if err != nil {
		return err
	}

	return err
}

func AddChunksToChromaCollection(ctx context.Context, collectionName string, chunks ...parser.Chunk) error {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	collection, err := client.GetCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	ids := make([]chroma.DocumentID, len(chunks))
	texts := make([]string, len(chunks))

	for i, doc := range chunks {
		ids[i] = chroma.DocumentID(doc.SHA256())
		texts[i] = doc.String()
	}

	err = collection.Add(ctx, chroma.WithIDs(ids...), chroma.WithTexts(texts...))
	if err != nil {
		return err
	}

	return nil
}

func SearchChromaCollection(ctx context.Context, collectionName string, topN int, queryStrings ...string) (chroma.QueryResult, error) {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	collection, err := client.GetCollection(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	queryResult, err := collection.Query(ctx, chroma.WithQueryTexts(queryStrings...), chroma.WithNResults(max(100, topN)))
	if err != nil {
		return nil, err
	}

	return queryResult, nil
}
