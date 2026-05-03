package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

func DecodeJSON[T any](ctx context.Context, client ChatClient, raw string, dest *T) error {
	if err := json.Unmarshal([]byte(raw), dest); err == nil {
		return nil
	}
	if client == nil {
		return fmt.Errorf("AI returned invalid JSON")
	}
	resp, err := client.Chat(ctx, ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "Repair the following content into valid strict JSON. Return only JSON."},
			{Role: "user", Content: raw},
		},
		Temperature: 0,
		Format:      "json",
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(resp.Content), dest); err != nil {
		return fmt.Errorf("AI returned invalid JSON after repair: %w", err)
	}
	return nil
}
