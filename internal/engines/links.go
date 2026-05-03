package engines

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/obsidian"
	"github.com/DrakeAFK/naudia/internal/proposals"
)

type LinkSuggestion struct {
	SourceNote string `json:"source_note"`
	TargetNote string `json:"target_note"`
	LineNumber int    `json:"line_number"`
	Reason     string `json:"reason"`
	Confidence string `json:"confidence"`
}

type LinksResult struct {
	Suggestions []LinkSuggestion    `json:"suggestions"`
	Proposal    *proposals.Proposal `json:"proposal,omitempty"`
}

func (r Runner) Links(ctx context.Context, noteFilter, folder string) (LinksResult, error) {
	notes, err := r.Store.ListNotes(ctx, r.VaultID)
	if err != nil {
		return LinksResult{}, err
	}
	var suggestions []LinkSuggestion
	var actions []proposals.ProposalAction
	for _, source := range notes {
		if noteFilter != "" && !strings.Contains(strings.ToLower(source.Path), strings.ToLower(noteFilter)) && !strings.Contains(strings.ToLower(source.Title), strings.ToLower(noteFilter)) {
			continue
		}
		if folder != "" && !strings.HasPrefix(source.Path, strings.Trim(folder, "/")+"/") {
			continue
		}
		content, hash, err := readVaultFile(r.Config.Vault.Path, source.Path)
		if err != nil {
			continue
		}
		lines := strings.Split(content, "\n")
		for _, target := range notes {
			if source.Path == target.Path || len(strings.TrimSpace(target.Title)) < 4 {
				continue
			}
			if strings.Contains(content, "[["+target.Title) || strings.Contains(content, target.Path) {
				continue
			}
			for i, line := range lines {
				if strings.Contains(line, target.Title) {
					old := line
					newLine := strings.Replace(line, target.Title, "[["+target.Title+"]]", 1)
					suggestions = append(suggestions, LinkSuggestion{
						SourceNote: source.Path,
						TargetNote: target.Path,
						LineNumber: i + 1,
						Reason:     "The source note mentions an existing note title without a wiki link.",
						Confidence: "high",
					})
					actions = append(actions, proposals.ProposalAction{
						ID:           fmt.Sprintf("link-%d", len(actions)+1),
						Kind:         proposals.ActionUpdateLines,
						Path:         source.Path,
						ExpectedHash: hash,
						LineEdits: []proposals.LineEdit{{
							LineNumber: i + 1,
							OldText:    old,
							NewText:    newLine,
						}},
					})
					break
				}
			}
			if len(actions) >= 10 {
				break
			}
		}
		if len(actions) >= 10 {
			break
		}
	}
	if aiSuggestions, err := r.aiLinks(ctx, suggestions); err == nil {
		suggestions = mergeLinkSuggestions(suggestions, aiSuggestions)
	}
	var proposal *proposals.Proposal
	if len(actions) > 0 {
		sourceNotes := make([]proposals.SourceNote, 0, len(suggestions))
		seen := map[string]bool{}
		for _, suggestion := range suggestions {
			if seen[suggestion.SourceNote] {
				continue
			}
			seen[suggestion.SourceNote] = true
			sourceNotes = append(sourceNotes, proposals.SourceNote{
				Path:        suggestion.SourceNote,
				ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, suggestion.SourceNote),
				Reason:      "Contains an unlinked mention of an existing note.",
			})
		}
		proposal = &proposals.Proposal{
			Type:        proposals.TypeLinkSuggestions,
			Title:       fmt.Sprintf("Add %d conservative missing links", len(actions)),
			Summary:     "Replace exact title mentions with wiki links. Each edit is line-scoped and stale-hash protected.",
			SourceNotes: sourceNotes,
			Actions:     actions,
			RiskLevel:   proposals.RiskLow,
			CreatedAt:   time.Now().UTC(),
		}
	}
	return LinksResult{Suggestions: suggestions, Proposal: proposal}, nil
}

func (r Runner) aiLinks(ctx context.Context, deterministic []LinkSuggestion) ([]LinkSuggestion, error) {
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil {
		return nil, fmt.Errorf("ollama unavailable")
	}
	pack, err := r.BuildContext(ctx, "missing links backlinks related notes orphan notes", "links")
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for _, s := range deterministic {
		fmt.Fprintf(&b, "- %s -> %s line %d: %s\n", s.SourceNote, s.TargetNote, s.LineNumber, s.Reason)
	}
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
			{Role: "user", Content: ai.LinksPrompt + "\n\nDeterministic suggestions:\n" + b.String() + "\nContext pack:\n" + contextpack.Render(pack)},
		},
		Temperature: 0.1,
		Format:      "json",
	})
	if err != nil {
		return nil, err
	}
	var out aiLinkOutput
	if err := ai.DecodeJSON(ctx, r.AI, resp.Content, &out); err != nil {
		return nil, err
	}
	var suggestions []LinkSuggestion
	for _, s := range out.Suggestions {
		if strings.TrimSpace(s.SourceNote) == "" || strings.TrimSpace(s.TargetNote) == "" {
			continue
		}
		suggestions = append(suggestions, LinkSuggestion{
			SourceNote: s.SourceNote,
			TargetNote: s.TargetNote,
			Reason:     nonEmpty(s.Reason, "AI suggested a conservative graph improvement from selected context."),
			Confidence: nonEmpty(s.Confidence, "medium"),
		})
	}
	return suggestions, nil
}

func mergeLinkSuggestions(base, extra []LinkSuggestion) []LinkSuggestion {
	seen := map[string]bool{}
	out := make([]LinkSuggestion, 0, len(base)+len(extra))
	for _, s := range append(base, extra...) {
		key := strings.ToLower(s.SourceNote + "->" + s.TargetNote)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func (r Runner) OrphanNotes(ctx context.Context) ([]string, error) {
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT n.path
		FROM notes n
		LEFT JOIN links out ON out.source_note_id = n.id AND out.resolved = 1
		LEFT JOIN links inc ON inc.target_note_id = n.id AND inc.resolved = 1
		WHERE n.vault_id = ?
		GROUP BY n.id
		HAVING COUNT(out.id) = 0 AND COUNT(inc.id) = 0
		ORDER BY n.path
	`, r.VaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func noteLine(path string, line int) string {
	return fmt.Sprintf("%s:%d", path, line)
}

func vaultPath(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

func fileHash(root, rel string) string {
	data, err := os.ReadFile(vaultPath(root, rel))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", data)
}
