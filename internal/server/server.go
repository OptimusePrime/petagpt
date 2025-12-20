package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OptimusePrime/petagpt/internal/db"
	"github.com/OptimusePrime/petagpt/internal/index"
	"github.com/OptimusePrime/petagpt/internal/sqlc"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/spf13/viper"
)

type SendMessageRequest struct {
	SessionID   string `json:"session_id"`
	UserMessage string `json:"user_message"`
}

func StartServer(idxName string, topN int) error {
	router := gin.Default()

	router.Use(cors.Default())

	router.POST("/chat/create", func(c *gin.Context) {
		handleCreateConversation(c, idxName, topN)
	})

	router.POST("/chat/send", func(c *gin.Context) {
		handleSendConversationMessage(c, idxName, topN)
	})

	router.Run(":7030")

	return nil
}

func Retrieval(ctx context.Context, queries []string, idxName string, topN int) string {
	var chunks string

	for _, q := range queries {
		result, err := index.SearchIndex(ctx, idxName, q, topN)
		if err != nil {
			return ""
		}
		//fmt.Println(result)

		for _, doc := range result.Documents {
			chunks += fmt.Sprintf("<document>\n%s\n</document>\n", doc.Content)
		}
	}

	return chunks
}

type CreateConversationRequest struct {
	SessionID string `json:"session_id"`
}

func handleCreateConversation(c *gin.Context, idxName string, topN int) {
	req := new(CreateConversationRequest)
	err := c.Bind(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to parse request body: %s", err.Error()),
		})
		return
	}

	queries := sqlc.New(db.MainDB)

	_, err = queries.CreateConversation(context.Background(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create conversation: %s", err.Error()),
		})
		return
	}

	c.Status(http.StatusOK)
}

func handleSendConversationMessage(c *gin.Context, idxName string, topN int) {
	client := openai.NewClient(
		option.WithAPIKey(viper.GetString("main_llm.api_key")),
		option.WithBaseURL(viper.GetString("main_llm.api_base")),
	)

	ctx := context.Background()

	req := new(SendMessageRequest)
	err := c.Bind(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to parse request body: %s", err.Error()),
		})
	}

	queries := sqlc.New(db.MainDB)

	conversation, err := queries.GetConversationBySessionID(context.Background(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get conversation: %s", err.Error()),
		})
		return
	}

	go func() {
		userAgent := c.Request.Header.Get("User-Agent")
		ipv4Addr := c.ClientIP()

		_, err = queries.CreateMessage(context.Background(), sqlc.CreateMessageParams{
			ConversationID: conversation.SessionID,
			Content:        req.UserMessage,
			UserAgent: sql.NullString{
				String: userAgent,
				Valid:  true,
			},
			Ipv4Addr: sql.NullString{
				String: ipv4Addr,
				Valid:  true,
			},
			Role: "user",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to save user message in database: %s", err.Error()),
			})
			return
		}
	}()

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(viper.GetString("main_llm.system_prompt")),
	}

	// for _, msg := range req.Messages {
	// 	if msg.Role == "assistant" {
	// 		msgs = append(msgs, openai.AssistantMessage(msg.Content))
	// 	} else {
	// 		msgs = append(msgs, openai.UserMessage(msg.Content))
	// 	}
	// }

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
						Description: openai.String("Find information about V. gimnazija and related subjects. You may enter mulitple queries at once. Use the tool when you believe you need additional information to answer the question."),
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
		ctx, params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Errorf("failed to create the chat completer: %s", err.Error()),
		})
		return
	}

	//fmt.Println(chatCompl.Choices[0].RawJSON())
	toolCalls := chatCompl.Choices[0].Message.ToolCalls
	if len(toolCalls) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"response": chatCompl.Choices[0].Message.Content,
		})
		return
	}
	fmt.Println(toolCalls)

	params.Messages = append(params.Messages, chatCompl.Choices[0].Message.ToParam())
	for _, toolCall := range toolCalls {
		if toolCall.Function.Name == "retrieval" {
			var args struct {
				Queries []string `json:"queries"`
			}
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}
			//fmt.Println(args)
			queries := args.Queries

			chunks := Retrieval(ctx, queries, idxName, topN)
			fmt.Printf("%+v", chunks)

			params.Messages = append(params.Messages, openai.ToolMessage(chunks, toolCall.ID))
		}
	}

	//params.ToolChoice = openai.ChatCompletionToolChoiceOptionAutoNone

	chatCompl, err = client.Chat.Completions.New(
		ctx, params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chatCompl == nil || len(chatCompl.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Empty response from LLM"})
		return
	}

	assistantMsg := chatCompl.Choices[0].Message.Content
	queries.CreateMessage(context.Background(), sqlc.CreateMessageParams{
		ConversationID: conversation.SessionID,
		Content:        assistantMsg,
		Role:           "assistant",
	})

	c.JSON(http.StatusOK, gin.H{
		"response": chatCompl.Choices[0].Message.Content,
	})
}
