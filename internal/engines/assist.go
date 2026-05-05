package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/obsidian"
	"github.com/DrakeAFK/naudia/internal/proposals"
)

type AssistResult struct {
	Answer            string              `json:"answer"`
	FollowUpQuestions []string            `json:"follow_up_questions,omitempty"`
	Context           contextpack.Pack    `json:"context"`
	Proposal          *proposals.Proposal `json:"proposal,omitempty"`
	Model             string              `json:"model,omitempty"`
}

func (r Runner) Assist(ctx context.Context, message string, history []ai.Message) (AssistResult, error) {
	pack, err := r.BuildContext(ctx, message, "ask")
	if err != nil {
		return AssistResult{}, err
	}
	if r.AI == nil {
		return AssistResult{Answer: fallbackContextAnswer(message, pack), Context: pack}, nil
	}
	messages := []ai.Message{{Role: "system", Content: ai.SystemPrompt + "\n\n" + ai.AssistantPrompt}}
	messages = append(messages, trimHistory(history, 10)...)
	messages = append(messages, ai.Message{Role: "user", Content: "Context:\n" + contextpack.Render(pack) + "\n\nUser message:\n" + message})
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages:    messages,
		Temperature: 0.1,
		Format:      "json",
	})
	if err != nil {
		if len(pack.Items) == 0 {
			return AssistResult{Answer: "I could not reach the local model and I do not have matching indexed context for that yet.", Context: pack}, nil
		}
		return AssistResult{Answer: fallbackContextAnswer(message, pack), Context: pack}, nil
	}
	var out aiAssistantOutput
	if err := ai.DecodeJSON(ctx, r.AI, resp.Content, &out); err != nil {
		ask, askErr := r.Ask(ctx, message)
		if askErr != nil {
			return AssistResult{}, askErr
		}
		return AssistResult{Answer: ask.Answer, Context: ask.Context, Model: ask.Model}, nil
	}
	result := AssistResult{
		Answer:            strings.TrimSpace(out.Answer),
		FollowUpQuestions: cleanStrings(out.FollowUpQuestions),
		Context:           pack,
		Model:             resp.Model,
	}
	if result.Answer == "" && len(result.FollowUpQuestions) > 0 {
		result.Answer = result.FollowUpQuestions[0]
	}
	proposal := r.assistantProposal(message, out, pack)
	if proposal != nil {
		result.Proposal = proposal
		if result.Answer == "" {
			result.Answer = "I prepared a reviewable proposal for that change."
		}
	}
	if result.Answer == "" {
		if len(pack.Items) == 0 {
			result.Answer = "I do not have enough indexed vault context to answer that. Run naudia scan, then try again."
		} else {
			result.Answer = fallbackContextAnswer(message, pack)
		}
	}
	return result, nil
}

func (r Runner) assistantProposal(message string, out aiAssistantOutput, pack contextpack.Pack) *proposals.Proposal {
	var actions []proposals.ProposalAction
	var sourceNotes []proposals.SourceNote
	for _, item := range pack.Items {
		if len(sourceNotes) >= 5 {
			break
		}
		sourceNotes = append(sourceNotes, proposals.SourceNote{
			Path:        item.NotePath,
			ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, item.NotePath),
			Reason:      item.Reason,
		})
	}
	for _, action := range out.Actions {
		kind := proposals.ActionKind(strings.ToLower(strings.TrimSpace(action.Kind)))
		if kind != proposals.ActionCreateNote && kind != proposals.ActionAppendToNote {
			continue
		}
		path := cleanAssistantPath(action.Path)
		content := strings.TrimSpace(action.Content)
		if path == "" || content == "" {
			continue
		}
		proposalAction := proposals.ProposalAction{
			ID:      fmt.Sprintf("assistant-action-%d", len(actions)+1),
			Kind:    kind,
			Path:    path,
			Content: content + "\n",
		}
		if kind == proposals.ActionAppendToNote {
			if _, hash, err := readVaultFile(r.Config.Vault.Path, path); err == nil {
				proposalAction.ExpectedHash = hash
			}
		}
		actions = append(actions, proposalAction)
	}
	if len(actions) == 0 {
		return nil
	}
	title := "Assistant note update"
	if len(actions) == 1 {
		switch actions[0].Kind {
		case proposals.ActionCreateNote:
			title = "Create " + actions[0].Path
		case proposals.ActionAppendToNote:
			title = "Append to " + actions[0].Path
		}
	}
	return &proposals.Proposal{
		Type:        proposals.TypeNoteUpdate,
		Title:       title,
		Summary:     "Prepared from an interactive Naudia request: " + truncateForSummary(message, 180),
		SourceNotes: sourceNotes,
		Actions:     actions,
		RiskLevel:   proposals.RiskLow,
		CreatedAt:   time.Now().UTC(),
	}
}

func (r Runner) NoteChangeProposal(kind proposals.ActionKind, path string, content string) (*proposals.Proposal, error) {
	if kind != proposals.ActionCreateNote && kind != proposals.ActionAppendToNote {
		return nil, fmt.Errorf("unsupported note action: %s", kind)
	}
	path = cleanAssistantPath(path)
	content = strings.TrimSpace(content)
	if path == "" {
		return nil, fmt.Errorf("note path is required")
	}
	if content == "" {
		return nil, fmt.Errorf("note content is required")
	}
	action := proposals.ProposalAction{
		ID:      "direct-note-change",
		Kind:    kind,
		Path:    path,
		Content: content + "\n",
	}
	if kind == proposals.ActionAppendToNote {
		if _, hash, err := readVaultFile(r.Config.Vault.Path, path); err == nil {
			action.ExpectedHash = hash
		}
	}
	title := "Create " + path
	typ := proposals.TypeNoteCreate
	if kind == proposals.ActionAppendToNote {
		title = "Append to " + path
		typ = proposals.TypeNoteUpdate
	}
	return &proposals.Proposal{
		Type:      typ,
		Title:     title,
		Summary:   "Prepared from a direct Naudia note command.",
		Actions:   []proposals.ProposalAction{action},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func trimHistory(history []ai.Message, max int) []ai.Message {
	if max <= 0 || len(history) <= max {
		return history
	}
	return history[len(history)-max:]
}

func cleanStrings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func cleanAssistantPath(raw string) string {
	path := strings.Trim(strings.TrimSpace(raw), "`\"'")
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "/")
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || path == "" || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
		return ""
	}
	if filepath.Ext(path) == "" {
		path += ".md"
	}
	return path
}

func truncateForSummary(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if max <= 0 || len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max-1]) + "."
}
