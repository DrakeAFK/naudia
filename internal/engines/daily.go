package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/contextpack"
	"github.com/drakeafk/naudia/internal/obsidian"
	"github.com/drakeafk/naudia/internal/proposals"
	"github.com/drakeafk/naudia/internal/vault"
)

type DailyResult struct {
	Date       string              `json:"date"`
	SourceNote string              `json:"source_note"`
	Review     string              `json:"review"`
	Context    contextpack.Pack    `json:"context"`
	Proposal   *proposals.Proposal `json:"proposal,omitempty"`
}

func (r Runner) Daily(ctx context.Context, dateText string, showContext bool) (DailyResult, error) {
	if dateText == "" {
		dateText = time.Now().Format(r.Config.Daily.DateFormat)
	}
	source := filepath.ToSlash(filepath.Join(r.Config.Daily.Folder, dateText+".md"))
	content, hash, err := readVaultFile(r.Config.Vault.Path, source)
	if err != nil {
		return DailyResult{}, err
	}
	note := vault.ParseNote(source, filepath.Join(r.Config.Vault.Path, filepath.FromSlash(source)), content)
	review := buildDailyReview(dateText, source, note)
	pack, err := r.BuildContext(ctx, dateText, "daily")
	if err != nil {
		return DailyResult{}, err
	}
	appendSection := "\n## Naudia Review\n\n" + review + "\n"
	proposal := &proposals.Proposal{
		Type:    proposals.TypeDailyDistillation,
		Title:   "Distill daily note " + dateText,
		Summary: "Append a sourced daily review with tasks, decisions, project updates, ideas, and carry-forward items.",
		SourceNotes: []proposals.SourceNote{{
			Path:        source,
			ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, source),
			Reason:      "Target daily note.",
		}},
		Actions: []proposals.ProposalAction{{
			ID:           "append-daily-review",
			Kind:         proposals.ActionAppendToNote,
			Path:         source,
			Content:      appendSection,
			ExpectedHash: hash,
		}},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	}
	return DailyResult{Date: dateText, SourceNote: source, Review: review, Context: pack, Proposal: proposal}, nil
}

func buildDailyReview(dateText, source string, note vault.Note) string {
	var tasks, decisions, updates, ideas, carry []string
	for _, task := range note.Tasks {
		mark := "[ ]"
		if task.Completed {
			mark = "[x]"
		}
		tasks = append(tasks, fmt.Sprintf("- %s %s", mark, task.Text))
		if !task.Completed {
			carry = append(carry, "- "+task.Text)
		}
	}
	for _, line := range strings.Split(note.Content, "\n") {
		trim := strings.TrimSpace(line)
		lower := strings.ToLower(trim)
		if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "- [") {
			continue
		}
		switch {
		case strings.Contains(lower, "decided") || strings.HasPrefix(lower, "decision:"):
			decisions = append(decisions, "- "+trim)
		case strings.Contains(lower, "project") || strings.Contains(lower, "shipped") || strings.Contains(lower, "built"):
			updates = append(updates, "- "+trim)
		case strings.Contains(lower, "idea") || strings.Contains(lower, "maybe"):
			ideas = append(ideas, "- "+trim)
		}
	}
	summary := firstMeaningfulLines(note.Content, 3)
	var b strings.Builder
	fmt.Fprintf(&b, "# Daily Review - %s\n\n", dateText)
	fmt.Fprintf(&b, "Source: %s\n\n", source)
	writeSection(&b, "Summary", summary)
	writeSection(&b, "Decisions", decisions)
	writeSection(&b, "Tasks", tasks)
	writeSection(&b, "Project Updates", updates)
	writeSection(&b, "Ideas Worth Keeping", ideas)
	writeSection(&b, "Notes to Create", []string{"- No permanent notes were proposed by deterministic review."})
	writeSection(&b, "Carry Forward", carry)
	return strings.TrimSpace(b.String()) + "\n"
}

func writeSection(b *strings.Builder, title string, lines []string) {
	fmt.Fprintf(b, "## %s\n", title)
	if len(lines) == 0 {
		b.WriteString("- Nothing explicit found in the source note.\n\n")
		return
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
}

func firstMeaningfulLines(content string, max int) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "---") || strings.Contains(trim, ":") && len(out) == 0 {
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		out = append(out, "- "+trim)
		if len(out) >= max {
			break
		}
	}
	return out
}
