package parser

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/spf13/viper"
)

func CreateChunkContext(ctx context.Context, documentContent string, chunkContent string) (error, string) {
	client := openai.NewClient(
		option.WithAPIKey(viper.GetString("context_llm.api_key")),
		option.WithBaseURL(viper.GetString("context_llm.api_base")),
	)

	r := strings.NewReplacer(
		"{{DOCUMENT}}", documentContent,
		"{{CHUNK}}", chunkContent,
	)

	replacedPrompt := r.Replace(viper.GetString("context_llm.prompt"))

	chatCompl, err := client.Chat.Completions.New(
		ctx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(replacedPrompt),
			},
			Model:       viper.GetString("context_llm.model"),
			Temperature: openai.Float(viper.GetFloat64("context_llm.temperature")),
			TopP:        openai.Float(viper.GetFloat64("context_llm.top_p")),
		},
	)
	if err != nil {
		return err, ""
	}

	chunkContext := chatCompl.Choices[0].Message.Content

	return nil, chunkContext
}
