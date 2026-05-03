package engines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/proposals"
)

type TaskItem struct {
	NotePath   string `json:"note_path"`
	Text       string `json:"text"`
	Completed  bool   `json:"completed"`
	LineNumber int    `json:"line_number"`
	Inferred   bool   `json:"inferred"`
}

func (r Runner) Tasks(ctx context.Context, project string) (Report, *proposals.Proposal, error) {
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT n.path, t.text, t.completed, COALESCE(t.line_number, 0), t.inferred
		FROM tasks t
		JOIN notes n ON n.id = t.note_id
		WHERE n.vault_id = ?
		ORDER BY n.path, t.line_number
	`, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	defer rows.Close()
	var tasks []TaskItem
	for rows.Next() {
		var t TaskItem
		var completed, inferred int
		if err := rows.Scan(&t.NotePath, &t.Text, &completed, &t.LineNumber, &inferred); err != nil {
			return Report{}, nil, err
		}
		t.Completed = completed == 1
		t.Inferred = inferred == 1
		if project == "" || strings.Contains(strings.ToLower(t.Text+" "+t.NotePath), strings.ToLower(project)) {
			tasks = append(tasks, t)
		}
	}
	report := Report{Title: "Tasks", Summary: fmt.Sprintf("Naudia found %d explicit Markdown tasks.", len(tasks))}
	for _, task := range tasks {
		state := "open"
		if task.Completed {
			state = "done"
		}
		report.Lines = append(report.Lines, fmt.Sprintf("%s [%s] %s", task.NotePath, state, task.Text))
	}
	if len(tasks) == 0 {
		return report, nil, rows.Err()
	}
	var b strings.Builder
	b.WriteString("# Task Inbox\n\n")
	for _, task := range tasks {
		if task.Completed {
			continue
		}
		fmt.Fprintf(&b, "- [ ] %s (source: [[%s]])\n", task.Text, strings.TrimSuffix(task.NotePath, ".md"))
	}
	proposal := &proposals.Proposal{
		Type:    proposals.TypeTaskExtraction,
		Title:   "Create task inbox from explicit tasks",
		Summary: "Create a reviewable task inbox sourced from existing Markdown checkboxes.",
		Actions: []proposals.ProposalAction{{
			ID:      "create-task-inbox",
			Kind:    proposals.ActionCreateNote,
			Path:    "Tasks/Inbox.md",
			Content: b.String(),
		}},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	}
	return report, proposal, rows.Err()
}
