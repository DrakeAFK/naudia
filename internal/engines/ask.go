package engines

import (
	"context"
	"strings"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/contextpack"
)

type AskResult struct {
	Answer  string           `json:"answer"`
	Context contextpack.Pack `json:"context"`
}

func (r Runner) Ask(ctx context.Context, question string) (AskResult, error) {
	pack, err := r.BuildContext(ctx, question, "ask")
	if err != nil {
		return AskResult{}, err
	}
	if r.AI != nil && len(pack.Items) > 0 {
		resp, err := r.AI.Chat(ctx, ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: ai.SystemPrompt},
				{Role: "user", Content: "Context:\n" + contextpack.Render(pack) + "\n\nQuestion:\n" + question + "\n\nAnswer with citations to source note paths."},
			},
			Temperature: 0.1,
		})
		if err == nil && strings.TrimSpace(resp.Content) != "" {
			return AskResult{Answer: resp.Content, Context: pack}, nil
		}
	}
	if len(pack.Items) == 0 {
		return AskResult{Answer: "I do not have enough indexed vault context to answer that. Run naudia scan, then try again.", Context: pack}, nil
	}
	var b strings.Builder
	b.WriteString("I found potentially relevant context, but Ollama was unavailable. Review these sources:\n")
	for _, item := range pack.Items {
		b.WriteString("- ")
		b.WriteString(item.NotePath)
		if item.Heading != "" {
			b.WriteString(" / ")
			b.WriteString(item.Heading)
		}
		b.WriteString("\n")
	}
	return AskResult{Answer: b.String(), Context: pack}, nil
}
