package engines

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/ai"
	"github.com/drakeafk/naudia/internal/contextpack"
	"github.com/drakeafk/naudia/internal/obsidian"
	"github.com/drakeafk/naudia/internal/proposals"
	"github.com/drakeafk/naudia/internal/util"
)

func (r Runner) Review(ctx context.Context, noAI bool, folder string) (Report, []*proposals.Proposal, error) {
	notes, err := r.Store.ListNotes(ctx, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	incoming := map[string]int{}
	outgoing := map[string]int{}
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT sn.path, COALESCE(tn.path, ''), l.resolved
		FROM links l
		JOIN notes sn ON sn.id = l.source_note_id
		LEFT JOIN notes tn ON tn.id = l.target_note_id
		WHERE sn.vault_id = ?
	`, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	unresolved := 0
	for rows.Next() {
		var src, target string
		var resolved int
		if err := rows.Scan(&src, &target, &resolved); err != nil {
			_ = rows.Close()
			return Report{}, nil, err
		}
		outgoing[src]++
		if resolved == 1 && target != "" {
			incoming[target]++
		} else {
			unresolved++
		}
	}
	_ = rows.Close()
	noTags := 0
	rootNotes := 0
	orphans := 0
	for _, note := range notes {
		if !strings.Contains(note.Path, "/") {
			rootNotes++
		}
		if incoming[note.Path] == 0 && outgoing[note.Path] == 0 {
			orphans++
		}
		var tagCount int
		_ = r.Store.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE note_id = ?`, note.ID).Scan(&tagCount)
		if tagCount == 0 {
			noTags++
		}
	}
	openTasks := 0
	dailyWithTasks := map[string]bool{}
	taskRows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT n.path FROM tasks t JOIN notes n ON n.id = t.note_id
		WHERE n.vault_id = ? AND t.completed = 0
	`, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	for taskRows.Next() {
		var path string
		if err := taskRows.Scan(&path); err != nil {
			_ = taskRows.Close()
			return Report{}, nil, err
		}
		openTasks++
		if strings.HasPrefix(path, strings.Trim(r.Config.Daily.Folder, "/")+"/") {
			dailyWithTasks[path] = true
		}
	}
	_ = taskRows.Close()
	issues := []Issue{}
	if rootNotes > 0 {
		issues = append(issues, Issue{Category: "Structure", Severity: "medium", Description: fmt.Sprintf("%d notes are in the vault root.", rootNotes), SuggestedAction: "Review root-level notes and move durable project notes into folders."})
	}
	if orphans > 0 {
		issues = append(issues, Issue{Category: "Links", Severity: "low", Description: fmt.Sprintf("%d notes have no incoming or outgoing resolved links.", orphans), SuggestedAction: "Run naudia links to prepare conservative backlink proposals."})
	}
	if unresolved > 0 {
		issues = append(issues, Issue{Category: "Links", Severity: "medium", Description: fmt.Sprintf("%d links are unresolved.", unresolved), SuggestedAction: "Fix ambiguous or missing note links."})
	}
	if openTasks > 0 {
		issues = append(issues, Issue{Category: "Tasks", Severity: "medium", Description: fmt.Sprintf("%d open Markdown tasks were found.", openTasks), SuggestedAction: "Run naudia tasks to group tasks by project."})
	}
	if len(dailyWithTasks) > 0 {
		issues = append(issues, Issue{Category: "Daily Notes", Severity: "medium", Description: fmt.Sprintf("%d daily notes contain open tasks.", len(dailyWithTasks)), SuggestedAction: "Run naudia daily to distill and carry forward important items."})
	}
	if noTags > 0 {
		issues = append(issues, Issue{Category: "Metadata", Severity: "low", Description: fmt.Sprintf("%d notes have no tags.", noTags), SuggestedAction: "Add tags only where they create retrieval or review value."})
	}
	summary := fmt.Sprintf("Naudia reviewed %d notes and found %d maintenance signals.", len(notes), len(issues))
	report := Report{
		Title:   "Vault Review",
		Summary: summary,
		Issues:  issues,
		Lines: []string{
			fmt.Sprintf("Notes: %d", len(notes)),
			fmt.Sprintf("Root notes: %d", rootNotes),
			fmt.Sprintf("Orphan notes: %d", orphans),
			fmt.Sprintf("Unresolved links: %d", unresolved),
			fmt.Sprintf("Open tasks: %d", openTasks),
		},
	}
	if !noAI {
		if aiReport, err := r.aiReview(ctx, report); err == nil {
			if strings.TrimSpace(aiReport.Summary) != "" {
				report.Summary = aiReport.Summary
			}
			for _, issue := range aiReport.Issues {
				report.Issues = append(report.Issues, Issue{
					Category:        nonEmpty(issue.Category, "AI Review"),
					Severity:        nonEmpty(issue.Severity, "low"),
					Description:     issue.Description,
					SourceNotes:     issue.SourceNotes,
					SuggestedAction: issue.SuggestedAction,
				})
			}
		}
	}
	report.ReportPath = r.writeReport("vault-review", renderReportMarkdown(report))
	var generated []*proposals.Proposal
	content := renderReportMarkdown(report)
	reviewPath := filepath.ToSlash(filepath.Join("Reviews", fmt.Sprintf("Vault Review %s.md", time.Now().Format("2006-01-02"))))
	generated = append(generated, &proposals.Proposal{
		Type:    proposals.TypeVaultReview,
		Title:   "Create vault review note",
		Summary: "Create a durable review note sourced from the deterministic vault scan.",
		SourceNotes: []proposals.SourceNote{{
			Path:        ".naudia/reports/" + filepath.Base(report.ReportPath),
			ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, reviewPath),
			Reason:      "Deterministic vault review report.",
		}},
		Actions: []proposals.ProposalAction{{
			ID:      "create-review-note",
			Kind:    proposals.ActionCreateNote,
			Path:    reviewPath,
			Content: content,
		}},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	})
	return report, generated, nil
}

func (r Runner) aiReview(ctx context.Context, deterministic Report) (aiReviewOutput, error) {
	var out aiReviewOutput
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil {
		return out, fmt.Errorf("ollama unavailable")
	}
	pack, err := r.BuildContext(ctx, "vault review structure tasks links templates unresolved questions", "structure")
	if err != nil {
		return out, err
	}
	prompt := "Deterministic review:\n" + renderReportMarkdown(deterministic) + "\n\nContext pack:\n" + contextpack.Render(pack) + "\n\n" + ai.ReviewPrompt
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		Format:      "json",
	})
	if err != nil {
		return out, err
	}
	if err := ai.DecodeJSON(ctx, r.AI, resp.Content, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (r Runner) writeReport(prefix, content string) string {
	name := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02-150405"), prefix)
	path := filepath.Join(r.Config.Vault.Path, ".naudia", "reports", name)
	_ = util.WriteFileAtomic(path, []byte(content), 0o644)
	rel, _ := filepath.Rel(r.Config.Vault.Path, path)
	return filepath.ToSlash(rel)
}

func renderReportMarkdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%s\n\n", report.Title, report.Summary)
	if len(report.Issues) > 0 {
		b.WriteString("## Findings\n\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(&b, "- **%s** (%s): %s\n", issue.Category, issue.Severity, issue.Description)
			if issue.SuggestedAction != "" {
				fmt.Fprintf(&b, "  - Suggested action: %s\n", issue.SuggestedAction)
			}
		}
	}
	return b.String()
}

func readVaultFile(root, rel string) (string, string, error) {
	full, err := util.ResolveInside(root, rel)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", "", err
	}
	return string(data), util.SHA256Bytes(data), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
