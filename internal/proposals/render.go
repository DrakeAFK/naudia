package proposals

import (
	"fmt"
	"strings"

	"github.com/drakeafk/naudia/internal/db"
)

func RenderList(records []db.ProposalRecord) string {
	if len(records) == 0 {
		return "No proposals found."
	}
	var b strings.Builder
	for _, p := range records {
		fmt.Fprintf(&b, "%3d  %-42s %-8s %s\n", p.ID, truncate(p.Title, 42), p.Status, p.Type)
	}
	return strings.TrimRight(b.String(), "\n")
}

func RenderDetails(p Proposal, rec db.ProposalRecord) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Proposal %d: %s\n\n", rec.ID, p.Title)
	fmt.Fprintf(&b, "Status: %s\n", rec.Status)
	fmt.Fprintf(&b, "Type: %s\n", p.Type)
	fmt.Fprintf(&b, "Risk: %s\n\n", p.RiskLevel)
	if p.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", p.Summary)
	}
	if len(p.SourceNotes) > 0 {
		b.WriteString("Sources:\n")
		for _, source := range p.SourceNotes {
			fmt.Fprintf(&b, "- %s", source.Path)
			if source.Reason != "" {
				fmt.Fprintf(&b, ": %s", source.Reason)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Actions:\n")
	for _, action := range p.Actions {
		fmt.Fprintf(&b, "- %s %s", action.Kind, action.Path)
		if action.NewPath != "" {
			fmt.Fprintf(&b, " -> %s", action.NewPath)
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(rec.PatchText) != "" {
		b.WriteString("\nPatch:\n")
		b.WriteString(rec.PatchText)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "."
}
