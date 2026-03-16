package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kyungw00k/mnemo/internal/config"
)

// ExtractedMemory holds a single memory fact extracted from text by an LLM.
type ExtractedMemory struct {
	Category string `json:"category"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

// ExtractService calls an OpenAI-compatible LLM to extract memories from text.
type ExtractService struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewExtractService creates a new ExtractService from config.
func NewExtractService(cfg *config.Config) *ExtractService {
	return &ExtractService{
		baseURL: strings.TrimRight(cfg.ExtractLLMBaseURL, "/"),
		apiKey:  cfg.ExtractLLMAPIKey,
		model:   cfg.ExtractLLMModel,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Extract calls the chat completions endpoint and parses the JSON response.
// Returns an empty slice on parse failure rather than an error.
func (s *ExtractService) Extract(ctx context.Context, text string) ([]ExtractedMemory, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type responseFormat struct {
		Type string `json:"type"`
	}
	type requestBody struct {
		Model          string         `json:"model"`
		Messages       []message      `json:"messages"`
		ResponseFormat responseFormat `json:"response_format"`
	}

	reqBody := requestBody{
		Model: s.model,
		Messages: []message{
			{
				Role: "system",
				Content: "Extract important facts from the following text as a JSON array. " +
					"Each item must have: category (string, e.g. project/decision/config/preference), " +
					"key (short identifier), value (the fact). " +
					"Return ONLY valid JSON array, no markdown.",
			},
			{
				Role:    "user",
				Content: text,
			},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal extract request: %w", err)
	}

	url := s.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create extract request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("extract LLM call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("extract LLM returned %d: %s", resp.StatusCode, body)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("extract: failed to decode LLM response: %v", err)
		return []ExtractedMemory{}, nil
	}

	if len(apiResp.Choices) == 0 {
		log.Printf("extract: LLM returned no choices")
		return []ExtractedMemory{}, nil
	}

	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)

	// Try to parse as a bare JSON array first.
	var memories []ExtractedMemory
	if err := json.Unmarshal([]byte(content), &memories); err == nil {
		return memories, nil
	}

	// Try to parse as a JSON object wrapping the array.
	var wrapper struct {
		Memories []ExtractedMemory `json:"memories"`
		Items    []ExtractedMemory `json:"items"`
		Data     []ExtractedMemory `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err == nil {
		if len(wrapper.Memories) > 0 {
			return wrapper.Memories, nil
		}
		if len(wrapper.Items) > 0 {
			return wrapper.Items, nil
		}
		if len(wrapper.Data) > 0 {
			return wrapper.Data, nil
		}
	}

	log.Printf("extract: could not parse LLM response as JSON array: %s", content)
	return []ExtractedMemory{}, nil
}
