package engines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/contextpack"
	"github.com/drakeafk/naudia/internal/proposals"
)

type ProjectResult struct {
	Name     string              `json:"name"`
	Context  contextpack.Pack    `json:"context"`
	Proposal *proposals.Proposal `json:"proposal,omitempty"`
}

func (r Runner) Project(ctx context.Context, name string) (ProjectResult, error) {
	pack, err := r.BuildContext(ctx, name, "project")
	if err != nil {
		return ProjectResult{}, err
	}
	folder := "Projects/" + safeName(name)
	contextText := contextpack.Render(pack)
	readme := fmt.Sprintf("# %s\n\n## Current Understanding\n\nThis project memory was compiled from selected vault context. Review the sources before applying further edits.\n\n## Sources\n\n%s\n", name, sourceList(pack))
	plan := fmt.Sprintf("# %s Plan\n\n## Context\n\n%s\n", name, contextText)
	todo := fmt.Sprintf("# %s TODO\n\n%s\n", name, tasksFromContext(pack))
	decisions := fmt.Sprintf("# %s Decisions\n\n%s\n", name, decisionsFromContext(pack))
	questions := fmt.Sprintf("# %s Questions\n\n%s\n", name, questionsFromContext(pack))
	files := map[string]string{
		folder + "/README.md":    readme,
		folder + "/PLAN.md":      plan,
		folder + "/TODO.md":      todo,
		folder + "/DECISIONS.md": decisions,
		folder + "/QUESTIONS.md": questions,
		folder + "/CONTEXT.md":   "# " + name + " Context\n\n" + contextText + "\n",
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
	for _, item := range pack.Items {
		proposal.SourceNotes = append(proposal.SourceNotes, proposals.SourceNote{Path: item.NotePath, ObsidianURI: item.ObsidianURI, Reason: item.Reason})
	}
	return ProjectResult{Name: name, Context: pack, Proposal: proposal}, nil
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
