package engines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/proposals"
)

type ProjectResult struct {
	Name     string              `json:"name"`
	Context  contextpack.Pack    `json:"context"`
	Proposal *proposals.Proposal `json:"proposal,omitempty"`
}

type ProjectOptions struct {
	Folder   string
	Generate []string
}

func (r Runner) Project(ctx context.Context, name string) (ProjectResult, error) {
	return r.ProjectWithOptions(ctx, name, ProjectOptions{})
}

func (r Runner) ProjectWithOptions(ctx context.Context, name string, opts ProjectOptions) (ProjectResult, error) {
	pack, err := r.BuildContext(ctx, name, "project")
	if err != nil {
		return ProjectResult{}, err
	}
	folder := "Projects/" + safeName(name)
	if strings.TrimSpace(opts.Folder) != "" {
		folder = strings.Trim(strings.ReplaceAll(opts.Folder, "\\", "/"), "/")
	}
	files := r.projectFilesFromAI(ctx, name, folder, pack)
	if len(files) == 0 {
		files = deterministicProjectFiles(name, folder, pack)
	}
	if filtered := filterProjectFiles(files, opts.Generate); len(filtered) > 0 {
		files = filtered
	}
	var actions []proposals.ProposalAction
	for path, content := range files {
		actions = append(actions, proposals.ProposalAction{
			ID:      "create-" + strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(path, folder+"/"), ".md", "")),
			Kind:    proposals.ActionCreateNote,
			Path:    path,
			Content: content,
		})
	}
	proposal := &proposals.Proposal{
		Type:      proposals.TypeProjectCompile,
		Title:     "Compile project memory for " + name,
		Summary:   "Create durable project files from a conservative context pack.",
		Actions:   actions,
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	}
	seenSources := map[string]bool{}
	for _, item := range pack.Items {
		if seenSources[item.NotePath] {
			continue
		}
		seenSources[item.NotePath] = true
		proposal.SourceNotes = append(proposal.SourceNotes, proposals.SourceNote{Path: item.NotePath, ObsidianURI: item.ObsidianURI, Reason: item.Reason})
	}
	return ProjectResult{Name: name, Context: pack, Proposal: proposal}, nil
}

func filterProjectFiles(files map[string]string, generate []string) map[string]string {
	if len(generate) == 0 {
		return files
	}
	allowed := map[string]bool{}
	for _, item := range generate {
		item = strings.ToLower(strings.TrimSpace(item))
		item = strings.TrimSuffix(item, ".md")
		if item != "" {
			allowed[item] = true
		}
	}
	if len(allowed) == 0 {
		return files
	}
	out := map[string]string{}
	for path, content := range files {
		base := strings.ToLower(strings.TrimSuffix(filepathBase(path), ".md"))
		if allowed[base] {
			out[path] = content
		}
	}
	return out
}

func filepathBase(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func (r Runner) projectFilesFromAI(ctx context.Context, name, folder string, pack contextpack.Pack) map[string]string {
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil || len(pack.Items) == 0 {
		return nil
	}
	resp, err := r.AI.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: ai.SystemPrompt},
			{Role: "user", Content: ai.ProjectPrompt + "\n\nProject: " + name + "\nContext pack:\n" + contextpack.Render(pack)},
		},
		Temperature: 0.1,
		Format:      "json",
	})
	if err != nil {
		return nil
	}
	var out aiProjectOutput
	if err := ai.DecodeJSON(ctx, r.AI, resp.Content, &out); err != nil {
		return nil
	}
	files := map[string]string{}
	for _, file := range out.SuggestedFiles {
		content := strings.TrimSpace(file.Content)
		if content == "" {
			continue
		}
		path := sanitizeProjectFilePath(folder, file.Path)
		files[path] = content + "\n"
	}
	if len(files) > 0 {
		if _, ok := files[folder+"/CONTEXT.md"]; !ok {
			files[folder+"/CONTEXT.md"] = "# " + name + " Context\n\n" + contextpack.Render(pack) + "\n"
		}
		return files
	}
	return nil
}

func deterministicProjectFiles(name, folder string, pack contextpack.Pack) map[string]string {
	contextText := contextpack.Render(pack)
	readme := fmt.Sprintf("# %s\n\n## Current Understanding\n\nThis project memory was compiled from selected vault context. Review the sources before applying further edits.\n\n## Sources\n\n%s\n", name, sourceList(pack))
	plan := fmt.Sprintf("# %s Plan\n\n## Context\n\n%s\n", name, contextText)
	todo := fmt.Sprintf("# %s TODO\n\n%s\n", name, tasksFromContext(pack))
	decisions := fmt.Sprintf("# %s Decisions\n\n%s\n", name, decisionsFromContext(pack))
	questions := fmt.Sprintf("# %s Questions\n\n%s\n", name, questionsFromContext(pack))
	return map[string]string{
		folder + "/README.md":    readme,
		folder + "/PLAN.md":      plan,
		folder + "/TODO.md":      todo,
		folder + "/DECISIONS.md": decisions,
		folder + "/QUESTIONS.md": questions,
		folder + "/CHANGELOG.md": "# " + name + " Changelog\n\n- Created project memory index from selected vault context.\n",
		folder + "/CONTEXT.md":   "# " + name + " Context\n\n" + contextText + "\n",
	}
}

func sanitizeProjectFilePath(folder, suggested string) string {
	suggested = strings.TrimSpace(strings.ReplaceAll(suggested, "\\", "/"))
	suggested = strings.TrimLeft(suggested, "/")
	if suggested == "" {
		return folder + "/CONTEXT.md"
	}
	if strings.Contains(suggested, "..") {
		suggested = strings.ReplaceAll(suggested, "..", "")
	}
	if strings.HasPrefix(suggested, folder+"/") {
		return suggested
	}
	base := strings.TrimPrefix(suggested, "Projects/")
	if strings.Contains(base, "/") {
		parts := strings.Split(base, "/")
		base = parts[len(parts)-1]
	}
	if !strings.HasSuffix(strings.ToLower(base), ".md") {
		base += ".md"
	}
	return folder + "/" + base
}

func sourceList(pack contextpack.Pack) string {
	if len(pack.Items) == 0 {
		return "- No strong source context was selected.\n"
	}
	var b strings.Builder
	seen := map[string]bool{}
	for _, item := range pack.Items {
		if seen[item.NotePath] {
			continue
		}
		seen[item.NotePath] = true
		fmt.Fprintf(&b, "- %s (%s)\n", item.NotePath, item.RetrievalMethod)
	}
	return b.String()
}

func tasksFromContext(pack contextpack.Pack) string {
	var b strings.Builder
	for _, item := range pack.Items {
		for _, line := range strings.Split(item.Excerpt, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "- [ ]") {
				fmt.Fprintf(&b, "%s (source: [[%s]])\n", trim, strings.TrimSuffix(item.NotePath, ".md"))
			}
		}
	}
	if b.Len() == 0 {
		return "- [ ] Review selected source notes and add concrete next actions.\n"
	}
	return b.String()
}

func decisionsFromContext(pack contextpack.Pack) string {
	return extractLines(pack, []string{"decided", "decision:"}, "No explicit decisions were selected in the context pack.")
}

func questionsFromContext(pack contextpack.Pack) string {
	return extractLines(pack, []string{"?", "question:"}, "No explicit open questions were selected in the context pack.")
}

func extractLines(pack contextpack.Pack, needles []string, empty string) string {
	var b strings.Builder
	for _, item := range pack.Items {
		for _, line := range strings.Split(item.Excerpt, "\n") {
			trim := strings.TrimSpace(line)
			lower := strings.ToLower(trim)
			for _, needle := range needles {
				if strings.Contains(lower, needle) {
					fmt.Fprintf(&b, "- %s (source: [[%s]])\n", trim, strings.TrimSuffix(item.NotePath, ".md"))
					break
				}
			}
		}
	}
	if b.Len() == 0 {
		return "- " + empty + "\n"
	}
	return b.String()
}
