package index

import (
	"context"
	"errors"
	"fmt"

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
		return fmt.Errorf("failed to create chroma client: %w", err)
	}

	_, err = client.CreateCollection(ctx, name, chroma.WithEmbeddingFunctionCreate(vllmEf))
	if err != nil {
		return fmt.Errorf("failed to create chroma collection: %w", err)
	}

	return nil
}

func DeleteChromaCollection(ctx context.Context, name string) error {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
	if err != nil {
		return fmt.Errorf("failed to create chroma client: %w", err)
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
		return fmt.Errorf("failed to create chroma client: %w", err)
	}

	collection, err := client.GetCollection(ctx, name, chroma.WithEmbeddingFunctionGet(vllmEf))
	if err != nil {
		return fmt.Errorf("failed to get chroma collection: %w", err)
	}

	err = collection.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete chroma collection: %w", err)
	}

	return nil
}

func RemoveChunksFromChromaCollection(ctx context.Context, collectionName string, chunksIDs []chroma.DocumentID) error {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
	defer func() {
		err = errors.Join(err, client.Close())
	}()

	vllmEf, err := vllm.NewVLLMEmbeddingFunctionFromOptions(
		vllm.WithModel(viper.GetString("embedding_service.model")),
		vllm.WithBaseURL(viper.GetString("embedding_service.base_url")),
		vllm.WithAPIKey(viper.GetString("embedding_service.api_key")),
	)
	if err != nil {
		return fmt.Errorf("failed to create vLLM embedding function: %w", err)
	}

	collection, err := client.GetCollection(ctx, collectionName, chroma.WithEmbeddingFunctionGet(vllmEf))
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	err = collection.Delete(ctx, chroma.WithIDsDelete(chunksIDs...))
	if err != nil {
		return fmt.Errorf("failed to delete chunks from collection: %w", err)
	}

	return nil
}

func AddChunksToChromaCollection(ctx context.Context, collectionName string, chunks ...parser.Chunk) error {
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(viper.GetString("chroma.base_url")))
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

	collection, err := client.GetCollection(ctx, collectionName, chroma.WithEmbeddingFunctionGet(vllmEf))
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
	//defer func() {
	//	err = errors.Join(err, fmt.Errorf("failed closing Chroma client: %w", client.Close()))
	//}()

	vllmEf, err := vllm.NewVLLMEmbeddingFunctionFromOptions(
		vllm.WithModel("Qwen/Qwen3-Embedding-4B"),
		vllm.WithBaseURL(viper.GetString("embedding_service.base_url")),
		vllm.WithAPIKey(viper.GetString("embedding_service.api_key")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed creating vLLM embedding function: %w", err)
	}

	collection, err := client.GetCollection(ctx, collectionName, chroma.WithEmbeddingFunctionGet(vllmEf))
	if err != nil {
		return nil, err
	}

	queryResult, err := collection.Query(ctx, chroma.WithQueryTexts(queryStrings...), chroma.WithNResults(max(100, topN)))
	if err != nil {
		return nil, err
	}

	return queryResult, nil
}
