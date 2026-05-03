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
	"github.com/DrakeAFK/naudia/internal/util"
)

func (r Runner) Review(ctx context.Context, noAI bool, folder string) (Report, []*proposals.Proposal, error) {
	notes, err := r.Store.ListNotes(ctx, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	if folder = strings.Trim(strings.ReplaceAll(folder, "\\", "/"), "/"); folder != "" {
		filtered := notes[:0]
		for _, note := range notes {
			if strings.HasPrefix(note.Path, folder+"/") || note.Path == folder || strings.HasPrefix(note.Path, folder) {
				filtered = append(filtered, note)
			}
		}
		notes = filtered
	}
	noteSet := map[string]bool{}
	for _, note := range notes {
		noteSet[note.Path] = true
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
		if len(noteSet) > 0 && !noteSet[src] && !noteSet[target] {
			continue
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
		if len(noteSet) > 0 && !noteSet[path] {
			continue
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
		Details: []ReportDetail{
			{Label: "Review mode", Value: "deterministic only"},
			{Label: "AI findings", Value: "disabled by --no-ai"},
		},
		Lines: []string{
			fmt.Sprintf("Notes: %d", len(notes)),
			fmt.Sprintf("Root notes: %d", rootNotes),
			fmt.Sprintf("Orphan notes: %d", orphans),
			fmt.Sprintf("Unresolved links: %d", unresolved),
			fmt.Sprintf("Open tasks: %d", openTasks),
		},
	}
	var generated []*proposals.Proposal
	if !noAI {
		report.Details = []ReportDetail{
			{Label: "Review mode", Value: "deterministic plus local AI"},
		}
		if aiReport, err := r.aiReview(ctx, report); err == nil {
			addedAIIssues := 0
			repeatedAIIssues := 0
			for _, issue := range aiReport.Issues {
				candidate := Issue{
					Category:        nonEmpty(issue.Category, "AI Review"),
					Severity:        nonEmpty(issue.Severity, "low"),
					Description:     issue.Description,
					SourceNotes:     issue.SourceNotes,
					SuggestedAction: issue.SuggestedAction,
				}
				if strings.TrimSpace(candidate.Description) == "" {
					continue
				}
				if duplicateIssue(report.Issues, candidate) {
					repeatedAIIssues++
					continue
				}
				report.Issues = append(report.Issues, candidate)
				addedAIIssues++
			}
			if addedAIIssues > 0 && strings.TrimSpace(aiReport.Summary) != "" {
				report.Summary = aiReport.Summary
			}
			generatedAI := r.proposalsFromAIReview(aiReport)
			if len(generatedAI) > 0 {
				generated = append(generated, generatedAI...)
			}
			report.Details = append(report.Details,
				ReportDetail{Label: "AI findings", Value: aiFindingsDetail(addedAIIssues, repeatedAIIssues)},
				ReportDetail{Label: "AI proposals", Value: fmt.Sprintf("%d prepared", len(generatedAI))},
			)
		} else {
			report.Details = append(report.Details, ReportDetail{Label: "AI findings", Value: "unavailable; deterministic review used"})
		}
	}
	report.ReportPath = r.writeReport("vault-review", renderReportMarkdown(report))
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

func aiFindingsDetail(added, repeated int) string {
	if added == 0 && repeated == 0 {
		return "no new sourced findings"
	}
	if added == 0 {
		return fmt.Sprintf("no new sourced findings; %d repeated deterministic findings ignored", repeated)
	}
	if repeated == 0 {
		return fmt.Sprintf("%d new sourced findings", added)
	}
	return fmt.Sprintf("%d new sourced findings; %d repeated deterministic findings ignored", added, repeated)
}

func (r Runner) proposalsFromAIReview(out aiReviewOutput) []*proposals.Proposal {
	var generated []*proposals.Proposal
	for i, p := range out.Proposals {
		title := strings.TrimSpace(p.Title)
		if title == "" {
			continue
		}
		risk := proposals.RiskLevel(strings.ToLower(strings.TrimSpace(p.RiskLevel)))
		if risk != proposals.RiskLow && risk != proposals.RiskMedium && risk != proposals.RiskHigh {
			risk = proposals.RiskLow
		}
		var sources []proposals.SourceNote
		for _, path := range p.SourceNotes {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			sources = append(sources, proposals.SourceNote{
				Path:        path,
				ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, path),
				Reason:      "AI review cited this source.",
			})
		}
		content := renderAIProposalNote(title, p.Type, p.Summary, p.SourceNotes, risk)
		slug := slugify(title)
		if slug == "" {
			slug = fmt.Sprintf("ai-review-suggestion-%d", i+1)
		}
		generated = append(generated, &proposals.Proposal{
			Type:        proposals.TypeVaultReview,
			Title:       title,
			Summary:     nonEmpty(p.Summary, "Create a sourced review note from AI-assisted vault review."),
			SourceNotes: sources,
			Actions: []proposals.ProposalAction{{
				ID:      fmt.Sprintf("create-ai-review-suggestion-%d", i+1),
				Kind:    proposals.ActionCreateNote,
				Path:    filepath.ToSlash(filepath.Join("Reviews", "AI Suggestions", slug+".md")),
				Content: content,
			}},
			RiskLevel:            risk,
			RequiresConfirmation: risk == proposals.RiskHigh,
			CreatedAt:            time.Now().UTC(),
		})
	}
	return generated
}

func duplicateIssue(existing []Issue, candidate Issue) bool {
	cat := strings.ToLower(strings.TrimSpace(candidate.Category))
	desc := normalizeIssueText(candidate.Description)
	for _, issue := range existing {
		if strings.ToLower(strings.TrimSpace(issue.Category)) != cat {
			continue
		}
		existingDesc := normalizeIssueText(issue.Description)
		if existingDesc == "" || desc == "" {
			continue
		}
		if desc == existingDesc || strings.Contains(desc, existingDesc) || strings.Contains(existingDesc, desc) {
			return true
		}
	}
	return false
}

func normalizeIssueText(s string) string {
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		":", " ",
		";", " ",
		"!", " ",
		"?", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
	)
	s = strings.ToLower(replacer.Replace(s))
	s = strings.ReplaceAll(s, "suggested action", " ")
	return strings.Join(strings.Fields(s), " ")
}

func renderAIProposalNote(title, typ, summary string, sources []string, risk proposals.RiskLevel) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	if typ != "" {
		fmt.Fprintf(&b, "Type: %s\n", typ)
	}
	fmt.Fprintf(&b, "Risk: %s\n\n", risk)
	if strings.TrimSpace(summary) != "" {
		fmt.Fprintf(&b, "## Summary\n\n%s\n\n", summary)
	}
	if len(sources) > 0 {
		b.WriteString("## Sources\n\n")
		for _, source := range sources {
			if strings.TrimSpace(source) != "" {
				fmt.Fprintf(&b, "- %s\n", source)
			}
		}
	}
	return b.String()
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
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
	if len(report.Details) > 0 {
		b.WriteString("## Details\n\n")
		for _, detail := range report.Details {
			if strings.TrimSpace(detail.Label) != "" || strings.TrimSpace(detail.Value) != "" {
				fmt.Fprintf(&b, "- **%s**: %s\n", detail.Label, detail.Value)
			}
		}
		b.WriteString("\n")
	}
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
