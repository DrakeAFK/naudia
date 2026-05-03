package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/ai"
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

type DailyOptions struct {
	Date                 string
	Week                 bool
	CreatePermanentNotes bool
	MoveTasks            bool
	ShowContext          bool
}

func (r Runner) Daily(ctx context.Context, dateText string, showContext bool) (DailyResult, error) {
	return r.DailyWithOptions(ctx, DailyOptions{Date: dateText, ShowContext: showContext})
}

func (r Runner) DailyWithOptions(ctx context.Context, opts DailyOptions) (DailyResult, error) {
	dateText := opts.Date
	if dateText == "" {
		dateText = time.Now().Format(r.Config.Daily.DateFormat)
	}
	if opts.Week {
		return r.dailyWeek(ctx, dateText, opts)
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
	if aiReview, err := r.aiDaily(ctx, dateText, source, content, pack); err == nil && strings.TrimSpace(aiReview) != "" {
		review = aiReview
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
	if opts.CreatePermanentNotes {
		proposal.Actions = append(proposal.Actions, permanentNoteActionsFromDaily(dateText, note)...)
	}
	if opts.MoveTasks {
		taskAction, err := r.dailyTaskAction(dateText, note)
		if err != nil {
			return DailyResult{}, err
		}
		if taskAction != nil {
			proposal.Actions = append(proposal.Actions, *taskAction)
		}
	}
	return DailyResult{Date: dateText, SourceNote: source, Review: review, Context: pack, Proposal: proposal}, nil
}

func (r Runner) dailyWeek(ctx context.Context, endDateText string, opts DailyOptions) (DailyResult, error) {
	end, err := time.Parse(r.Config.Daily.DateFormat, endDateText)
	if err != nil {
		return DailyResult{}, err
	}
	var reviews []string
	var sources []proposals.SourceNote
	var actions []proposals.ProposalAction
	for i := 6; i >= 0; i-- {
		date := end.AddDate(0, 0, -i).Format(r.Config.Daily.DateFormat)
		source := filepath.ToSlash(filepath.Join(r.Config.Daily.Folder, date+".md"))
		content, hash, err := readVaultFile(r.Config.Vault.Path, source)
		if err != nil {
			continue
		}
		note := vault.ParseNote(source, filepath.Join(r.Config.Vault.Path, filepath.FromSlash(source)), content)
		review := buildDailyReview(date, source, note)
		reviews = append(reviews, review)
		sources = append(sources, proposals.SourceNote{Path: source, ObsidianURI: obsidian.BuildOpenNoteURI(r.Config.Vault.Name, source), Reason: "Daily note included in weekly distillation."})
		actions = append(actions, proposals.ProposalAction{
			ID:           "append-daily-review-" + date,
			Kind:         proposals.ActionAppendToNote,
			Path:         source,
			Content:      "\n## Naudia Review\n\n" + review + "\n",
			ExpectedHash: hash,
		})
		if opts.CreatePermanentNotes {
			actions = append(actions, permanentNoteActionsFromDaily(date, note)...)
		}
		if opts.MoveTasks {
			if taskAction, err := r.dailyTaskAction(date, note); err == nil && taskAction != nil {
				actions = append(actions, *taskAction)
			}
		}
	}
	if len(actions) == 0 {
		return DailyResult{}, fmt.Errorf("no daily notes found for week ending %s", endDateText)
	}
	pack, err := r.BuildContext(ctx, endDateText, "daily")
	if err != nil {
		return DailyResult{}, err
	}
	weekly := "# Weekly Daily Review - week ending " + endDateText + "\n\n" + strings.Join(reviews, "\n\n---\n\n")
	proposal := &proposals.Proposal{
		Type:        proposals.TypeDailyDistillation,
		Title:       "Distill daily notes for week ending " + endDateText,
		Summary:     "Append sourced daily reviews for each daily note found in the selected week.",
		SourceNotes: sources,
		Actions:     actions,
		RiskLevel:   proposals.RiskMedium,
		CreatedAt:   time.Now().UTC(),
	}
	return DailyResult{Date: endDateText, SourceNote: strings.TrimSuffix(r.Config.Daily.Folder, "/") + "/*", Review: weekly, Context: pack, Proposal: proposal}, nil
}

func permanentNoteActionsFromDaily(dateText string, note vault.Note) []proposals.ProposalAction {
	var actions []proposals.ProposalAction
	for i, line := range strings.Split(note.Content, "\n") {
		trim := strings.TrimSpace(line)
		lower := strings.ToLower(trim)
		if !strings.Contains(lower, "idea") && !strings.Contains(lower, "decided") && !strings.Contains(lower, "decision:") {
			continue
		}
		title := strings.Trim(strings.TrimPrefix(strings.TrimPrefix(trim, "Idea:"), "Decision:"), " .")
		if title == "" {
			title = fmt.Sprintf("Daily note insight %s %d", dateText, i+1)
		}
		if len(title) > 60 {
			title = title[:60]
		}
		path := filepath.ToSlash(filepath.Join("Notes", slugify(title)+".md"))
		content := fmt.Sprintf("# %s\n\n%s\n\nSource: [[%s]]\n", title, trim, strings.TrimSuffix(note.Path, ".md"))
		actions = append(actions, proposals.ProposalAction{
			ID:      fmt.Sprintf("create-permanent-note-%d", len(actions)+1),
			Kind:    proposals.ActionCreateNote,
			Path:    path,
			Content: content,
		})
		if len(actions) >= 5 {
			break
		}
	}
	return actions
}

func (r Runner) dailyTaskAction(dateText string, note vault.Note) (*proposals.ProposalAction, error) {
	var taskLines []string
	for _, task := range note.Tasks {
		if task.Completed {
			continue
		}
		taskLines = append(taskLines, fmt.Sprintf("- [ ] %s (source: [[%s]])", task.Text, strings.TrimSuffix(note.Path, ".md")))
	}
	if len(taskLines) == 0 {
		return nil, nil
	}
	target := "Tasks/Inbox.md"
	_, hash, err := readVaultFile(r.Config.Vault.Path, target)
	if err != nil {
		content := "# Task Inbox\n\n" + strings.Join(taskLines, "\n") + "\n"
		return &proposals.ProposalAction{ID: "create-task-inbox-" + dateText, Kind: proposals.ActionCreateNote, Path: target, Content: content}, nil
	}
	content := "\n## " + dateText + "\n" + strings.Join(taskLines, "\n") + "\n"
	return &proposals.ProposalAction{ID: "append-task-inbox-" + dateText, Kind: proposals.ActionAppendToNote, Path: target, Content: content, ExpectedHash: hash}, nil
}

func (r Runner) aiDaily(ctx context.Context, dateText, source, content string, pack contextpack.Pack) (string, error) {
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil {
		return "", fmt.Errorf("ollama unavailable")
	}
	prompt := ai.DailyPrompt + "\n\nDaily note path: " + source + "\nDaily note content:\n" + content + "\n\nContext pack:\n" + contextpack.Render(pack)
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		Format:      "json",
	})
	if err != nil {
		return "", err
	}
	var out aiDailyOutput
	if err := ai.DecodeJSON(ctx, r.AI, resp.Content, &out); err != nil {
		return "", err
	}
	if out.Date == "" {
		out.Date = dateText
	}
	if out.SourceNote == "" {
		out.SourceNote = source
	}
	return renderAIDaily(out), nil
}

func renderAIDaily(out aiDailyOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Daily Review - %s\n\n", out.Date)
	fmt.Fprintf(&b, "Source: %s\n\n", out.SourceNote)
	writeSection(&b, "Summary", bulletOrFallback([]string{out.Summary}, "No summary was produced from the provided context."))
	writeSection(&b, "Decisions", prefixBullets(out.Decisions))
	var tasks []string
	for _, task := range out.Tasks {
		if strings.TrimSpace(task.Text) == "" {
			continue
		}
		label := "explicit"
		if !task.Explicit {
			label = "inferred"
		}
		project := ""
		if task.ProjectGuess != "" {
			project = " [" + task.ProjectGuess + "]"
		}
		tasks = append(tasks, "- [ ] "+task.Text+" ("+label+")"+project)
	}
	writeSection(&b, "Tasks", tasks)
	var updates []string
	for _, update := range out.ProjectUpdates {
		line := strings.TrimSpace(update.Update)
		if line == "" {
			continue
		}
		if update.Project != "" {
			line = update.Project + ": " + line
		}
		updates = append(updates, "- "+line)
	}
	writeSection(&b, "Project Updates", updates)
	writeSection(&b, "Ideas Worth Keeping", prefixBullets(out.IdeasWorthKeeping))
	var notes []string
	for _, note := range out.NotesToCreate {
		if strings.TrimSpace(note.Title) == "" {
			continue
		}
		notes = append(notes, "- "+note.Title+": "+note.Reason)
	}
	writeSection(&b, "Notes to Create", notes)
	writeSection(&b, "Carry Forward", prefixBullets(out.CarryForward))
	return strings.TrimSpace(b.String()) + "\n"
}

func prefixBullets(items []string) []string {
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "- ") || strings.HasPrefix(item, "- [") {
			out = append(out, item)
		} else {
			out = append(out, "- "+item)
		}
	}
	return out
}

func bulletOrFallback(items []string, fallback string) []string {
	items = prefixBullets(items)
	if len(items) == 0 {
		return []string{"- " + fallback}
	}
	return items
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
		if strings.HasPrefix(trim, "- [") {
			continue
		}
		out = append(out, "- "+trim)
		if len(out) >= max {
			break
		}
	}
	return out
}
