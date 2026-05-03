package ai

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	Format      string    `json:"format,omitempty"`
}

type ChatResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

type Embedding struct {
	Vector []float64 `json:"vector"`
}

type ChatClient interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type EmbeddingClient interface {
	Embed(ctx context.Context, input []string) ([]Embedding, error)
}
