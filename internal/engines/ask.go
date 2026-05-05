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
	Model   string           `json:"model,omitempty"`
}

func (r Runner) Ask(ctx context.Context, question string) (AskResult, error) {
	pack, err := r.BuildContext(ctx, question, "ask")
	if err != nil {
		return AskResult{}, err
	}
	if r.AI != nil && len(pack.Items) > 0 {
		resp, err := r.AI.Chat(ctx, ai.ChatRequest{
			Messages: []ai.Message{
				{Role: "system", Content: ai.SystemPrompt + "\n\n" + ai.AskPrompt},
				{Role: "user", Content: "Context:\n" + contextpack.Render(pack) + "\n\nQuestion:\n" + question},
			},
			Temperature: 0.05,
		})
		if err == nil && strings.TrimSpace(resp.Content) != "" {
			answer := strings.TrimSpace(resp.Content)
			if contextRefusal(answer) {
				retry, retryErr := r.AI.Chat(ctx, ai.ChatRequest{
					Messages: []ai.Message{
						{Role: "system", Content: ai.SystemPrompt + "\n\n" + ai.AskPrompt},
						{Role: "user", Content: "The selected context is relevant. Re-read it and answer the supported part of the question. If something is missing, put that after the answer.\n\nContext:\n" + contextpack.Render(pack) + "\n\nQuestion:\n" + question},
					},
					Temperature: 0.05,
				})
				if retryErr == nil && strings.TrimSpace(retry.Content) != "" && !contextRefusal(retry.Content) {
					return AskResult{Answer: strings.TrimSpace(retry.Content), Context: pack, Model: retry.Model}, nil
				}
			} else {
				return AskResult{Answer: answer, Context: pack, Model: resp.Model}, nil
			}
		}
	}
	if len(pack.Items) == 0 {
		return AskResult{Answer: "I do not have enough indexed vault context to answer that. Run naudia scan, then try again.", Context: pack}, nil
	}
	return AskResult{Answer: fallbackContextAnswer(question, pack), Context: pack}, nil
}

func fallbackContextAnswer(question string, pack contextpack.Pack) string {
	var b strings.Builder
	b.WriteString("I found relevant vault context, but the local model did not produce a usable answer. The strongest sources are:\n")
	tokens := queryTokens(question)
	for i, item := range pack.Items {
		if i >= 5 {
			break
		}
		b.WriteString("- ")
		b.WriteString(item.NotePath)
		if item.Heading != "" {
			b.WriteString(" / ")
			b.WriteString(item.Heading)
		}
		lines := bestEvidenceLines(item.Excerpt, tokens, 2)
		if len(lines) > 0 {
			b.WriteString(": ")
			b.WriteString(strings.Join(lines, " "))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func contextRefusal(answer string) bool {
	lower := strings.ToLower(strings.TrimSpace(answer))
	if lower == "" {
		return false
	}
	phrases := []string{
		"not enough context",
		"insufficient context",
		"don't have enough context",
		"do not have enough context",
		"provided context does not",
		"provided context doesn't",
		"context is insufficient",
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func bestEvidenceLines(excerpt string, tokens []string, limit int) []string {
	if limit <= 0 {
		limit = 2
	}
	lines := strings.Split(excerpt, "\n")
	var picked []string
	for _, line := range lines {
		trim := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		for _, token := range tokens {
			if strings.Contains(lower, token) {
				picked = append(picked, trim)
				break
			}
		}
		if len(picked) >= limit {
			return picked
		}
	}
	for _, line := range lines {
		trim := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		picked = append(picked, trim)
		if len(picked) >= limit {
			break
		}
	}
	return picked
}
