package llm

import (
	"github.com/liyue201/tian-niu/pkg/shared"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func NewLLMClient(modelConf shared.ModelConfig) openai.Client {
	client := openai.NewClient(
		option.WithBaseURL(modelConf.BaseURL),
		option.WithAPIKey(modelConf.ApiKey),
		option.WithHeader("X-Title", "Tianniu"),
	)
	return client
}
