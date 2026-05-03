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

type TaskOptions struct {
	Project         string
	Today           bool
	Week            bool
	IncludeInferred bool
}

func (r Runner) Tasks(ctx context.Context, project string) (Report, *proposals.Proposal, error) {
	return r.TasksWithOptions(ctx, TaskOptions{Project: project})
}

func (r Runner) TasksWithOptions(ctx context.Context, opts TaskOptions) (Report, *proposals.Proposal, error) {
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
	cutoffPath := ""
	if opts.Today || opts.Week {
		days := 0
		if opts.Week {
			days = 6
		}
		cutoffPath = strings.Trim(r.Config.Daily.Folder, "/") + "/" + time.Now().AddDate(0, 0, -days).Format(r.Config.Daily.DateFormat)
	}
	for rows.Next() {
		var t TaskItem
		var completed, inferred int
		if err := rows.Scan(&t.NotePath, &t.Text, &completed, &t.LineNumber, &inferred); err != nil {
			return Report{}, nil, err
		}
		t.Completed = completed == 1
		t.Inferred = inferred == 1
		if cutoffPath != "" && !strings.HasPrefix(t.NotePath, strings.Trim(r.Config.Daily.Folder, "/")+"/") {
			continue
		}
		if cutoffPath != "" && t.NotePath < cutoffPath {
			continue
		}
		if opts.Project == "" || strings.Contains(strings.ToLower(t.Text+" "+t.NotePath), strings.ToLower(opts.Project)) {
			tasks = append(tasks, t)
		}
	}
	if opts.IncludeInferred {
		inferred, err := r.inferTasks(ctx, opts, cutoffPath)
		if err != nil {
			return Report{}, nil, err
		}
		tasks = append(tasks, inferred...)
	}
	explicitCount := 0
	inferredCount := 0
	for _, task := range tasks {
		if task.Inferred {
			inferredCount++
		} else {
			explicitCount++
		}
	}
	report := Report{Title: "Tasks", Summary: fmt.Sprintf("Naudia found %d explicit and %d inferred tasks.", explicitCount, inferredCount)}
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

func (r Runner) inferTasks(ctx context.Context, opts TaskOptions, cutoffPath string) ([]TaskItem, error) {
	chunks, err := r.Store.ListChunks(ctx, r.VaultID)
	if err != nil {
		return nil, err
	}
	var inferred []TaskItem
	for _, chunk := range chunks {
		if cutoffPath != "" && (!strings.HasPrefix(chunk.NotePath, strings.Trim(r.Config.Daily.Folder, "/")+"/") || chunk.NotePath < cutoffPath) {
			continue
		}
		if opts.Project != "" && !strings.Contains(strings.ToLower(chunk.NotePath+" "+chunk.Content), strings.ToLower(opts.Project)) {
			continue
		}
		for _, line := range strings.Split(chunk.Content, "\n") {
			trim := strings.TrimSpace(line)
			lower := strings.ToLower(trim)
			if trim == "" || strings.HasPrefix(trim, "- [") {
				continue
			}
			if strings.Contains(lower, "need to ") || strings.Contains(lower, "should ") || strings.Contains(lower, "todo:") || strings.Contains(lower, "follow up") {
				inferred = append(inferred, TaskItem{NotePath: chunk.NotePath, Text: strings.TrimPrefix(trim, "- "), Inferred: true})
			}
			if len(inferred) >= 50 {
				return inferred, nil
			}
		}
	}
	return inferred, nil
}
