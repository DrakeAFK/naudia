package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/proposals"
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
	for folder, count := range folders {
		report.Lines = append(report.Lines, fmt.Sprintf("%s: %d notes", folder, count))
	}
	if len(root) > 0 {
		report.Issues = append(report.Issues, Issue{Category: "Structure", Severity: "medium", Description: fmt.Sprintf("%d notes are at the vault root.", len(root)), SuggestedAction: "Move root capture notes into 00 Inbox after review."})
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

func (r Runner) Templates(ctx context.Context, folder string) (Report, *proposals.Proposal, error) {
	if folder == "" {
		folder = "Templates"
	}
	required := map[string]string{
		"Daily.md":    "# Daily Note\n\n## Notes\n\n## Tasks\n\n## Decisions\n",
		"Project.md":  "# Project\n\n## Goal\n\n## Context\n\n## Tasks\n\n## Decisions\n",
		"Decision.md": "# Decision\n\n## Context\n\n## Decision\n\n## Consequences\n",
	}
	report := Report{Title: "Templates", Summary: "Naudia checked expected starter templates."}
	var actions []proposals.ProposalAction
	for name, content := range required {
		path := filepath.ToSlash(filepath.Join(folder, name))
		if _, _, err := readVaultFile(r.Config.Vault.Path, path); err == nil {
			report.Lines = append(report.Lines, path+" exists")
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
