package longterm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPEmbeddingService_Embed(t *testing.T) {
	expectedVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected path /embeddings, got %s", r.URL.Path)
		}

		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("Expected model test-model, got %s", req.Model)
		}
		if req.Input != "test text" {
			t.Errorf("Expected input 'test text', got %s", req.Input)
		}

		resp := embeddingResponse{
			Object: "list",
			Data: []struct {
				Index     int       `json:"index"`
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
			}{
				{
					Index:     0,
					Object:    "embedding",
					Embedding: expectedVector,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer testServer.Close()

	service := NewHTTPEmbeddingService(HTTPEmbeddingConfig{
		APIKey:     "test-api-key",
		BaseURL:    testServer.URL,
		Model:      "test-model",
		Dimensions: 5,
	})

	result, err := service.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	t.Logf("Embed result: %v", result)

	if len(result) != len(expectedVector) {
		t.Errorf("Expected vector length %d, got %d", len(expectedVector), len(result))
	}

	for i, v := range expectedVector {
		if result[i] != v {
			t.Errorf("Expected vector[%d] = %f, got %f", i, v, result[i])
		}
	}
}

func TestHTTPEmbeddingService_EmbedEmptyResponse(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddingResponse{
			Object: "list",
			Data: []struct {
				Index     int       `json:"index"`
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
			}{},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer testServer.Close()

	service := NewHTTPEmbeddingService(HTTPEmbeddingConfig{
		APIKey:  "test-api-key",
		BaseURL: testServer.URL,
		Model:   "test-model",
	})

	_, err := service.Embed(context.Background(), "test text")
	if err == nil {
		t.Error("Expected error for empty response, got nil")
	}
}

func TestHTTPEmbeddingService_EmbedServerError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	service := NewHTTPEmbeddingService(HTTPEmbeddingConfig{
		APIKey:  "test-api-key",
		BaseURL: testServer.URL,
		Model:   "test-model",
	})

	_, err := service.Embed(context.Background(), "test text")
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestHTTPEmbeddingService_EmbedNetworkError(t *testing.T) {
	service := NewHTTPEmbeddingService(HTTPEmbeddingConfig{
		APIKey:  "test-api-key",
		BaseURL: "http://localhost:1",
		Model:   "test-model",
	})

	_, err := service.Embed(context.Background(), "test text")
	if err == nil {
		t.Error("Expected error for network error, got nil")
	}
}
