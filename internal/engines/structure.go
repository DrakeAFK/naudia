package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/proposals"
)

func (r Runner) Structure(ctx context.Context, propose bool) (Report, *proposals.Proposal, error) {
	notes, err := r.Store.ListNotes(ctx, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	folders := map[string]int{}
	var root []string
	for _, note := range notes {
		dir := filepath.ToSlash(filepath.Dir(note.Path))
		if dir == "." {
			root = append(root, note.Path)
			dir = "root"
		}
		folders[dir]++
	}
	report := Report{Title: "Structure", Summary: fmt.Sprintf("Naudia analyzed %d notes across %d folders.", len(notes), len(folders))}
	inconsistentNames := 0
	staleNotes := 0
	now := time.Now()
	for folder, count := range folders {
		report.Lines = append(report.Lines, fmt.Sprintf("%s: %d notes", folder, count))
	}
	for _, note := range notes {
		if hasInconsistentName(note.Path) {
			inconsistentNames++
		}
		if t, err := time.Parse(time.RFC3339, note.ModifiedAt); err == nil && now.Sub(t) > 180*24*time.Hour {
			staleNotes++
		}
	}
	if len(root) > 0 {
		report.Issues = append(report.Issues, Issue{Category: "Structure", Severity: "medium", Description: fmt.Sprintf("%d notes are at the vault root.", len(root)), SuggestedAction: "Move root capture notes into 00 Inbox after review."})
	}
	if inconsistentNames > 0 {
		report.Issues = append(report.Issues, Issue{Category: "Naming", Severity: "low", Description: fmt.Sprintf("%d notes use spaces, uppercase extensions, or mixed separators.", inconsistentNames), SuggestedAction: "Standardize names only where predictable automation matters; avoid bulk renames without review."})
	}
	if staleNotes > 0 {
		report.Issues = append(report.Issues, Issue{Category: "Lifecycle", Severity: "low", Description: fmt.Sprintf("%d notes have not changed in more than 180 days.", staleNotes), SuggestedAction: "Review whether stale notes should be archived, linked, or left untouched."})
	}
	mixed := r.mixedFolderIssues(ctx)
	report.Issues = append(report.Issues, mixed...)
	if aiStructure, err := r.aiStructure(ctx, report); err == nil {
		if strings.TrimSpace(aiStructure.Summary) != "" {
			report.Summary = aiStructure.Summary
		}
		for _, issue := range aiStructure.CurrentIssues {
			if strings.TrimSpace(issue) != "" {
				report.Issues = append(report.Issues, Issue{Category: "Structure", Severity: "low", Description: issue, SuggestedAction: "Review structure recommendations before creating move proposals."})
			}
		}
		for _, rec := range aiStructure.RecommendedStructure {
			if strings.TrimSpace(rec.Path) != "" {
				report.Lines = append(report.Lines, fmt.Sprintf("Recommended %s: %s", rec.Path, rec.Purpose))
			}
		}
	}
	if !propose || len(root) == 0 {
		return report, nil, nil
	}
	var actions []proposals.ProposalAction
	for _, path := range root {
		if len(actions) >= 10 {
			break
		}
		_, hash, err := readVaultFile(r.Config.Vault.Path, path)
		if err != nil {
			continue
		}
		actions = append(actions, proposals.ProposalAction{
			ID:           fmt.Sprintf("move-root-%d", len(actions)+1),
			Kind:         proposals.ActionMoveNote,
			Path:         path,
			NewPath:      filepath.ToSlash(filepath.Join("00 Inbox", path)),
			ExpectedHash: hash,
		})
	}
	proposal := &proposals.Proposal{
		Type:                 proposals.TypeStructureChange,
		Title:                "Move reviewed root notes into 00 Inbox",
		Summary:              "Move a small batch of root-level notes into 00 Inbox. Moves are high risk and require explicit confirmation.",
		Actions:              actions,
		RiskLevel:            proposals.RiskHigh,
		RequiresConfirmation: true,
		CreatedAt:            time.Now().UTC(),
	}
	return report, proposal, nil
}

func (r Runner) aiStructure(ctx context.Context, deterministic Report) (aiStructureOutput, error) {
	var out aiStructureOutput
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil {
		return out, fmt.Errorf("ollama unavailable")
	}
	pack, err := r.BuildContext(ctx, "vault folder structure templates projects areas resources archive", "structure")
	if err != nil {
		return out, err
	}
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
			{Role: "user", Content: ai.StructurePrompt + "\n\nDeterministic structure report:\n" + renderReportMarkdown(deterministic) + "\nContext pack:\n" + contextpack.Render(pack)},
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

type TemplateOptions struct {
	Folder      string
	ProjectOnly bool
	DailyOnly   bool
}

func (r Runner) Templates(ctx context.Context, folder string) (Report, *proposals.Proposal, error) {
	return r.TemplatesWithOptions(ctx, TemplateOptions{Folder: folder})
}

func (r Runner) TemplatesWithOptions(ctx context.Context, opts TemplateOptions) (Report, *proposals.Proposal, error) {
	folder := opts.Folder
	if folder == "" {
		folder = "Templates"
	}
	required := map[string]string{
		"Daily.md":    "# Daily Note\n\n## Notes\n\n## Tasks\n\n## Decisions\n",
		"Project.md":  "# Project\n\n## Goal\n\n## Context\n\n## Tasks\n\n## Decisions\n",
		"Decision.md": "# Decision\n\n## Context\n\n## Decision\n\n## Consequences\n",
	}
	if opts.ProjectOnly {
		required = map[string]string{"Project.md": required["Project.md"]}
	}
	if opts.DailyOnly {
		required = map[string]string{"Daily.md": required["Daily.md"]}
	}
	report := Report{Title: "Templates", Summary: "Naudia checked expected starter templates."}
	var actions []proposals.ProposalAction
	for name, content := range required {
		path := filepath.ToSlash(filepath.Join(folder, name))
		if existing, hash, err := readVaultFile(r.Config.Vault.Path, path); err == nil {
			report.Lines = append(report.Lines, path+" exists")
			missing := missingTemplateHeadings(name, existing)
			for _, heading := range missing {
				report.Issues = append(report.Issues, Issue{Category: "Templates", Severity: "low", Description: path + " is missing " + heading + ".", SuggestedAction: "Append the missing heading without rewriting the existing template."})
				actions = append(actions, proposals.ProposalAction{
					ID:           "append-" + strings.ToLower(strings.Trim(heading, "# ")),
					Kind:         proposals.ActionAppendToNote,
					Path:         path,
					Content:      "\n" + heading + "\n",
					ExpectedHash: hash,
				})
			}
			continue
		}
		report.Issues = append(report.Issues, Issue{Category: "Templates", Severity: "low", Description: path + " is missing.", SuggestedAction: "Create a small starter template."})
		actions = append(actions, proposals.ProposalAction{ID: "create-" + strings.TrimSuffix(strings.ToLower(name), ".md"), Kind: proposals.ActionCreateNote, Path: path, Content: content})
	}
	if len(actions) == 0 {
		return report, nil, nil
	}
	return report, &proposals.Proposal{
		Type:      proposals.TypeTemplateImprovement,
		Title:     "Create missing starter templates",
		Summary:   "Create only missing templates; existing templates are left untouched.",
		Actions:   actions,
		RiskLevel: proposals.RiskMedium,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func hasInconsistentName(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, "  ") || strings.Contains(base, "_") || filepath.Ext(base) != ".md"
}

func (r Runner) mixedFolderIssues(ctx context.Context) []Issue {
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT substr(n.path, 1, CASE WHEN instr(n.path, '/') = 0 THEN length(n.path) ELSE instr(n.path, '/') - 1 END) AS folder,
		       SUM(CASE WHEN t.id IS NOT NULL THEN 1 ELSE 0 END) AS task_notes,
		       SUM(CASE WHEN lower(n.path) LIKE '%template%' THEN 1 ELSE 0 END) AS template_notes,
		       COUNT(DISTINCT n.id) AS total
		FROM notes n
		LEFT JOIN tasks t ON t.note_id = n.id AND t.completed = 0
		WHERE n.vault_id = ?
		GROUP BY folder
	`, r.VaultID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var issues []Issue
	for rows.Next() {
		var folder string
		var taskNotes, templateNotes, total int
		if err := rows.Scan(&folder, &taskNotes, &templateNotes, &total); err != nil {
			return issues
		}
		if total >= 5 && taskNotes > 0 && templateNotes > 0 {
			issues = append(issues, Issue{Category: "Structure", Severity: "low", Description: fmt.Sprintf("%s mixes templates and task-heavy notes.", folder), SuggestedAction: "Consider separating reusable templates from active work notes."})
		}
	}
	return issues
}

func missingTemplateHeadings(name, content string) []string {
	expected := map[string][]string{
		"Daily.md":    {"## Notes", "## Tasks", "## Decisions"},
		"Project.md":  {"## Goal", "## Context", "## Tasks", "## Decisions"},
		"Decision.md": {"## Context", "## Decision", "## Consequences"},
	}
	var missing []string
	for _, heading := range expected[name] {
		if !strings.Contains(strings.ToLower(content), strings.ToLower(heading)) {
			missing = append(missing, heading)
		}
	}
	return missing
}
