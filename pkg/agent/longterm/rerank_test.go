package longterm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRerankService_Rerank(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/rerank" {
			t.Errorf("Expected path /rerank, got %s", r.URL.Path)
		}

		var req rerankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if req.Model != "test-rerank-model" {
			t.Errorf("Expected model test-rerank-model, got %s", req.Model)
		}
		if req.Query != "test query" {
			t.Errorf("Expected query 'test query', got %s", req.Query)
		}
		if len(req.Documents) != 3 {
			t.Errorf("Expected 3 documents, got %d", len(req.Documents))
		}

		resp := rerankResponse{
			ID: "test-id",
			Results: []struct {
				Document       string  `json:"document"`
				Index          int     `json:"index"`
				RelevanceScore float32 `json:"relevance_score"`
			}{
				{Document: req.Documents[2], Index: 2, RelevanceScore: 0.95},
				{Document: req.Documents[0], Index: 0, RelevanceScore: 0.85},
				{Document: req.Documents[1], Index: 1, RelevanceScore: 0.75},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer testServer.Close()

	service := NewHTTPRerankService(HTTPRerankConfig{
		APIKey:  "test-api-key",
		BaseURL: testServer.URL,
		Model:   "test-rerank-model",
	})

	candidates := []MemoryMatch{
		{Content: "doc1", Summary: "document 1 content"},
		{Content: "doc2", Summary: "document 2 content"},
		{Content: "doc3", Summary: "document 3 content"},
	}

	result, err := service.Rerank(context.Background(), "test query", candidates)
	if err != nil {
		t.Fatalf("Rerank returned error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}

	expectedOrder := []int{2, 0, 1}
	for i, expectedIdx := range expectedOrder {
		if result[i].Content != candidates[expectedIdx].Content {
			t.Errorf("Expected result[%d] to be candidate[%d], got different content", i, expectedIdx)
		}
	}
}

func TestHTTPRerankService_RerankEmptyCandidates(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not make API call with empty candidates")
	}))
	defer testServer.Close()

	service := NewHTTPRerankService(HTTPRerankConfig{
		APIKey:  "test-api-key",
		BaseURL: testServer.URL,
		Model:   "test-rerank-model",
	})

	result, err := service.Rerank(context.Background(), "test query", []MemoryMatch{})
	if err != nil {
		t.Errorf("Expected no error for empty candidates, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestHTTPRerankService_RerankServerError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	service := NewHTTPRerankService(HTTPRerankConfig{
		APIKey:  "test-api-key",
		BaseURL: testServer.URL,
		Model:   "test-rerank-model",
	})

	candidates := []MemoryMatch{
		{Content: "doc1", Summary: "document 1"},
	}

	_, err := service.Rerank(context.Background(), "test query", candidates)
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestHTTPRerankService_RerankNetworkError(t *testing.T) {
	service := NewHTTPRerankService(HTTPRerankConfig{
		APIKey:  "test-api-key",
		BaseURL: "http://localhost:1",
		Model:   "test-rerank-model",
	})

	candidates := []MemoryMatch{
		{Content: "doc1", Summary: "document 1"},
	}

	_, err := service.Rerank(context.Background(), "test query", candidates)
	if err == nil {
		t.Error("Expected error for network error, got nil")
	}
}
