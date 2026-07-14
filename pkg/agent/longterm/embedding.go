package longterm

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type HTTPEmbeddingConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
}

type HTTPEmbeddingService struct {
	client *resty.Client
	config HTTPEmbeddingConfig
}

func NewHTTPEmbeddingService(config HTTPEmbeddingConfig) *HTTPEmbeddingService {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetHeader("Authorization", "Bearer "+config.APIKey).
		SetHeader("Content-Type", "application/json")

	return &HTTPEmbeddingService{
		client: client,
		config: config,
	}
}

type embeddingRequest struct {
	Model      string `json:"model"`
	Input      string `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Index     int       `json:"index"`
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (s *HTTPEmbeddingService) Embed(ctx context.Context, text string) (Vector, error) {
	var resp embeddingResponse

	req := embeddingRequest{
		Model: s.config.Model,
		Input: text,
	}

	if s.config.Dimensions > 0 {
		req.Dimensions = s.config.Dimensions
	}

	r := s.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp)

	_, err := r.Post("/embeddings")
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return Vector(resp.Data[0].Embedding), nil
}
