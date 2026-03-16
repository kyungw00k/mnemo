package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyungw00k/mnemo/internal/config"
)

// EmbeddingService generates vector embeddings using an OpenAI-compatible API.
type EmbeddingService struct {
	baseURL    string
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

// NewEmbeddingService creates an EmbeddingService from the given configuration.
func NewEmbeddingService(cfg *config.Config) *EmbeddingService {
	return &EmbeddingService{
		baseURL:    strings.TrimRight(cfg.EmbeddingBaseURL, "/"),
		apiKey:     cfg.EmbeddingAPIKey,
		model:      cfg.EmbeddingModel,
		dimensions: cfg.EmbeddingDimensions,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed generates a vector embedding for the given text.
// Returns nil, nil for empty text. Returns nil, err on API failure.
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, nil
	}

	reqBody, err := json.Marshal(embeddingRequest{
		Model: s.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	// The baseURL already includes /v1 per convention.
	url := s.baseURL + "/embeddings"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d", resp.StatusCode)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding API returned empty data")
	}

	return result.Data[0].Embedding, nil
}
