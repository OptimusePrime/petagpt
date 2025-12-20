package safety

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/spf13/viper"
)

type SafetyLevel string

const SAFETY_LEVEL_NONE = "None"
const SAFETY_LEVEL_CONTROVERSIAL = "Controversial"
const SAFETY_LEVEL_UNSAFE = "Unsafe"

type MessageSafety struct {
	SafetyLevel    SafetyLevel
	SafetyCategories []string
}

func CheckUserMessageSafety(ctx context.Context, msg string) (MessageSafety, error) {
	client := openai.NewClient(
		option.WithAPIKey(viper.GetString("safety_classifier.api_key")),
		option.WithBaseURL(viper.GetString("safety_classifier.api_base")),
	)

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("safety_classifier.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(msg),
		},
	}

	chatCompl, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return MessageSafety{}, err
	}

	result := strings.Split(chatCompl.Choices[0].Message.Content, "\n")
	safetyLevel := strings.Split(result[0], ": ")[1]
	safetyCategories := strings.Split(strings.Split(result[1], ": ")[1], ", ")

	return MessageSafety{
		SafetyLevel:    SafetyLevel(safetyLevel),
		SafetyCategories: safetyCategories,
	}, nil
}

func CheckAssistantMessageSafety(ctx context.Context, msg string) (MessageSafety, error) {
	client := openai.NewClient(
		option.WithAPIKey(viper.GetString("safety_classifier.api_key")),
		option.WithBaseURL(viper.GetString("safety_classifier.api_base")),
	)

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("safety_classifier.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.AssistantMessage(msg),
		},
	}

	chatCompl, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return MessageSafety{}, err
	}

	result := strings.Split(chatCompl.Choices[0].Message.Content, "\n")
	safetyLevel := strings.Split(result[0], ": ")[1]
	safetyCategories := strings.Split(strings.Split(result[1], ": ")[1], ", ")

	return MessageSafety{
		SafetyLevel:    SafetyLevel(safetyLevel),
		SafetyCategories: safetyCategories,
	}, nil
}
