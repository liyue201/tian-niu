package shared

import "github.com/openai/openai-go/v3"

type OpenAIMessage = openai.ChatCompletionMessageParamUnion

// GetRoleName returns the role name from a message (without relying on GetRole())
func GetRoleName(message OpenAIMessage) string {
	if message.OfSystem != nil {
		return "system"
	}
	if message.OfUser != nil {
		return "user"
	}
	if message.OfAssistant != nil {
		return "assistant"
	}
	if message.OfTool != nil {
		return "tool"
	}
	if message.OfDeveloper != nil {
		return "developer"
	}
	if message.OfFunction != nil {
		return "function"
	}
	return "unknown"
}
