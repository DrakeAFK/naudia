package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	Host           string
	ChatModel      string
	EmbeddingModel string
	HTTP           *http.Client
}

func NewOllamaClient(host, chatModel, embeddingModel string) *OllamaClient {
	host = strings.TrimRight(host, "/")
	return &OllamaClient{
		Host:           host,
		ChatModel:      chatModel,
		EmbeddingModel: embeddingModel,
		HTTP:           &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Host+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Host+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, model.Name)
	}
	return models, nil
}

func (c *OllamaClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.ChatModel
	}
	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
		"stream":   false,
		"options": map[string]any{
			"temperature": req.Temperature,
		},
	}
	if req.Format != "" {
		body["format"] = req.Format
	}
	var payload struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Model string `json:"model"`
	}
	if err := c.postJSON(ctx, "/api/chat", body, &payload); err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{Content: payload.Message.Content, Model: payload.Model}, nil
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":  c.ChatModel,
		"prompt": prompt,
		"stream": false,
	}
	var payload struct {
		Response string `json:"response"`
	}
	if err := c.postJSON(ctx, "/api/generate", body, &payload); err != nil {
		return "", err
	}
	return payload.Response, nil
}

func (c *OllamaClient) Embed(ctx context.Context, input []string) ([]Embedding, error) {
	if len(input) == 0 {
		return nil, nil
	}
	body := map[string]any{
		"model": c.EmbeddingModel,
		"input": input,
	}
	var embedPayload struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	err := c.postJSON(ctx, "/api/embed", body, &embedPayload)
	if err == nil && len(embedPayload.Embeddings) == len(input) {
		out := make([]Embedding, 0, len(embedPayload.Embeddings))
		for _, vec := range embedPayload.Embeddings {
			out = append(out, Embedding{Vector: vec})
		}
		return out, nil
	}
	var out []Embedding
	for _, text := range input {
		legacyBody := map[string]any{"model": c.EmbeddingModel, "prompt": text}
		var legacy struct {
			Embedding []float64 `json:"embedding"`
		}
		if legacyErr := c.postJSON(ctx, "/api/embeddings", legacyBody, &legacy); legacyErr != nil {
			if err != nil {
				return nil, err
			}
			return nil, legacyErr
		}
		out = append(out, Embedding{Vector: legacy.Embedding})
	}
	return out, nil
}

func (c *OllamaClient) postJSON(ctx context.Context, path string, body any, dest any) error {
	if c.Host == "" {
		return errors.New("ollama host is empty")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
