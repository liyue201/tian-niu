package longterm

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type HTTPRerankConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type HTTPRerankService struct {
	client *resty.Client
	config HTTPRerankConfig
}

func NewHTTPRerankService(config HTTPRerankConfig) *HTTPRerankService {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetHeader("Authorization", "Bearer "+config.APIKey).
		SetHeader("Content-Type", "application/json")

	return &HTTPRerankService{
		client: client,
		config: config,
	}
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type rerankResponse struct {
	ID      string `json:"id"`
	Results []struct {
		Document       string  `json:"document"`
		Index          int     `json:"index"`
		RelevanceScore float32 `json:"relevance_score"`
	} `json:"results"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (s *HTTPRerankService) Rerank(ctx context.Context, query string, candidates []MemoryMatch) ([]MemoryMatch, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	documents := make([]string, len(candidates))
	for i, match := range candidates {
		documents[i] = match.Summary
	}

	req := rerankRequest{
		Model:     s.config.Model,
		Query:     query,
		Documents: documents,
		TopN:      len(candidates),
	}

	var resp rerankResponse
	r := s.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&resp)

	_, err := r.Post("/rerank")
	if err != nil {
		return nil, fmt.Errorf("failed to call rerank API: %w", err)
	}

	result := make([]MemoryMatch, len(resp.Results))
	for i, item := range resp.Results {
		result[i] = candidates[item.Index]
		result[i].Score = item.RelevanceScore
	}

	return result, nil
}
