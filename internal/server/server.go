package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/spf13/viper"
)

type Message struct {
	Role    string
	Content string
}

type MessageRequest struct {
	Messages []Message
}

func StartServer(idxName string, topN int) error {
	router := gin.Default()

	router.POST("/chat", func(c *gin.Context) {
		client := openai.NewClient(
			option.WithAPIKey(viper.GetString("main_llm.api_key")),
			option.WithBaseURL(viper.GetString("main_llm.api_base")),
		)

		req := new(MessageRequest)
		err := c.Bind(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("failed to parse request body: %s", err.Error()),
			})
		}

		msgs := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(viper.GetString("main_llm.system_prompt")),
		}

		for _, msg := range req.Messages {
			if msg.Role == "assistant" {
				msgs = append(msgs, openai.AssistantMessage(msg.Content))
			} else {
				msgs = append(msgs, openai.UserMessage(msg.Content))
			}
		}

		params := openai.ChatCompletionNewParams{
			Messages:        msgs,
			Model:           viper.GetString("main_llm.model"),
			Temperature:     openai.Float(viper.GetFloat64("main_llm.temperature")),
			TopP:            openai.Float(viper.GetFloat64("main_llm.top_p")),
			ReasoningEffort: openai.ReasoningEffort(viper.GetString("main_llm.reasoning_effort")),
			Tools: []openai.ChatCompletionToolUnionParam{
				{
					OfFunction: &openai.ChatCompletionFunctionToolParam{
						Function: openai.FunctionDefinitionParam{
							Name:        "retrieval",
							Description: openai.String("Find information about V. gimnazija and related subjects. You may enter multiple queries at once, a maximum of four. Use the tool when you are not 100% sure you know the answer to the user's question or believe you need additional information to answer the question."),
							Parameters: openai.FunctionParameters{
								"type": "object",
								"properties": map[string]any{
									"queries": map[string]any{
										"type": "array",
										"items": map[string]any{
											"type": "string",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		chatCompl, err := client.Chat.Completions.New(
			context.Background(), params,
		)

		toolCalls := chatCompl.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"response": chatCompl.Choices[0].Message.Content,
			})
			return
		}

		for _, toolCall := range toolCalls {
			if toolCall.Function.Name == "retrieval" {
				var args map[string]interface{}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				if err != nil {
					panic(err)
				}
				queries := args["queries"].([]string)

				chunks := Retrieval(context.Background(), queries, idxName, topN)

				params.Messages = append(params.Messages, openai.ToolMessage(chunks, toolCall.ID))
			}
		}

		chatCompl, err = client.Chat.Completions.New(
			context.Background(), params,
		)
		c.JSON(http.StatusOK, gin.H{
			"response": chatCompl.Choices[0].Message.Content,
		})
	})

	return nil
}

func Retrieval(ctx context.Context, queries []string, idxName string, topN int) string {
	var chunks string

	for _, q := range queries {
		result, err := index.SearchIndex(ctx, idxName, q, topN)
		if err != nil {
			return ""
		}

		for _, doc := range result.Documents {
			chunks += fmt.Sprintf("<document>\n%s\n</document>\n", doc.Content)
		}
	}

	return chunks
}
