package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/engines"
	"github.com/DrakeAFK/naudia/internal/proposals"
	"github.com/DrakeAFK/naudia/internal/util"
)

var (
	proposalPathRequestRE = regexp.MustCompile(`(?i)\b(?:update|change|set)\s+proposal\s+([0-9]+)\b.*\b(?:file\s+path|path)\b.*\bto(?:\s+be)?\s+(.+)$`)
	proposalIDRE          = regexp.MustCompile(`(?i)\bproposal\s+#?([0-9]+)\b`)
)

func runNaudiaOperation(ctx context.Context, r engines.Runner, pm proposals.Manager, message string) (engines.AssistResult, bool, error) {
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)
	if match := proposalPathRequestRE.FindStringSubmatch(trimmed); len(match) == 3 {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return engines.AssistResult{}, true, err
		}
		answer, err := updateProposalActionPath(ctx, r, pm, id, match[2])
		return engines.AssistResult{Answer: answer, Context: contextpack.Pack{}}, true, err
	}
	if id, ok := parseNaturalShowProposal(lower); ok {
		p, rec, err := pm.Load(ctx, id)
		if err != nil {
			return engines.AssistResult{}, true, err
		}
		return engines.AssistResult{Answer: proposals.RenderDetails(*p, rec), Context: contextpack.Pack{}}, true, nil
	}
	if id, ok := parseProposalID(trimmed); ok && isProposalExplanationRequest(lower) {
		answer, err := proposalExplanation(ctx, pm, id)
		return engines.AssistResult{Answer: answer, Context: contextpack.Pack{}}, true, err
	}
	if isImplicitProposalExplanationRequest(lower) {
		answer, ok, err := explainOnlyPendingProposal(ctx, pm)
		return engines.AssistResult{Answer: answer, Context: contextpack.Pack{}}, ok, err
	}
	if isProposalListRequest(lower) {
		answer, err := pendingProposalAnswer(ctx, pm)
		return engines.AssistResult{Answer: answer, Context: contextpack.Pack{}}, true, err
	}
	return engines.AssistResult{}, false, nil
}

func isProposalListRequest(lower string) bool {
	if !strings.Contains(lower, "proposal") {
		return false
	}
	if strings.Contains(lower, "update proposal") || strings.Contains(lower, "change proposal") || strings.Contains(lower, "set proposal") {
		return false
	}
	return strings.Contains(lower, "current") ||
		strings.Contains(lower, "pending") ||
		strings.Contains(lower, "open") ||
		strings.Contains(lower, "what are") ||
		strings.Contains(lower, "list")
}

func parseNaturalShowProposal(lower string) (int64, bool) {
	fields := strings.Fields(lower)
	if len(fields) < 3 {
		return 0, false
	}
	if fields[0] != "show" && fields[0] != "view" && fields[0] != "review" {
		return 0, false
	}
	for i := 1; i < len(fields)-1; i++ {
		if fields[i] == "proposal" {
			id, err := strconv.ParseInt(strings.Trim(fields[i+1], "#., "), 10, 64)
			return id, err == nil
		}
	}
	return 0, false
}

func parseProposalID(message string) (int64, bool) {
	match := proposalIDRE.FindStringSubmatch(message)
	if len(match) != 2 {
		return 0, false
	}
	id, err := strconv.ParseInt(match[1], 10, 64)
	return id, err == nil
}

func isProposalExplanationRequest(lower string) bool {
	if !strings.Contains(lower, "proposal") {
		return false
	}
	if strings.Contains(lower, "update proposal") || strings.Contains(lower, "change proposal") || strings.Contains(lower, "set proposal") {
		return false
	}
	return strings.Contains(lower, "about") ||
		strings.Contains(lower, "mean") ||
		strings.Contains(lower, "meaning") ||
		strings.Contains(lower, "explain") ||
		strings.Contains(lower, "what is") ||
		strings.Contains(lower, "what's") ||
		strings.Contains(lower, "what does") ||
		strings.Contains(lower, "why")
}

func isImplicitProposalExplanationRequest(lower string) bool {
	if !strings.Contains(lower, "proposal") {
		return false
	}
	if strings.Contains(lower, "proposals") && (strings.Contains(lower, "current") || strings.Contains(lower, "pending") || strings.Contains(lower, "list")) {
		return false
	}
	return strings.Contains(lower, "what does") ||
		strings.Contains(lower, "what do") ||
		strings.Contains(lower, "what is") ||
		strings.Contains(lower, "what's") ||
		strings.Contains(lower, "mean") ||
		strings.Contains(lower, "about") ||
		strings.Contains(lower, "explain")
}

func explainOnlyPendingProposal(ctx context.Context, pm proposals.Manager) (string, bool, error) {
	records, err := pm.Store.ListProposals(ctx, pm.VaultID, "pending", false)
	if err != nil {
		return "", true, err
	}
	if len(records) != 1 {
		if len(records) == 0 {
			return "There are no pending proposals to explain.", true, nil
		}
		return fmt.Sprintf("There are %d pending proposals. Which one should I explain? Use `what is proposal <id> about` or `/show <id>`.", len(records)), true, nil
	}
	answer, err := proposalExplanation(ctx, pm, records[0].ID)
	return answer, true, err
}

func pendingProposalAnswer(ctx context.Context, pm proposals.Manager) (string, error) {
	records, err := pm.Store.ListProposals(ctx, pm.VaultID, "pending", false)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "No pending Naudia proposals.", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Pending Naudia proposals (%d):\n", len(records))
	limit := len(records)
	if limit > 20 {
		limit = 20
	}
	for _, rec := range records[:limit] {
		fmt.Fprintf(&b, "- #%d %s (%s)", rec.ID, rec.Title, rec.Type)
		if strings.TrimSpace(rec.Summary) != "" {
			fmt.Fprintf(&b, ": %s", rec.Summary)
		}
		b.WriteString("\n")
	}
	if len(records) > limit {
		fmt.Fprintf(&b, "- ...%d more\n", len(records)-limit)
	}
	b.WriteString("\nUse `/show <id>`, `/apply <id>`, or `/reject <id>`.")
	return strings.TrimSpace(b.String()), nil
}

func proposalExplanation(ctx context.Context, pm proposals.Manager, id int64) (string, error) {
	p, rec, err := pm.Load(ctx, id)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Proposal #%d is `%s`.\n\n", rec.ID, p.Title)
	if strings.TrimSpace(p.Summary) != "" {
		fmt.Fprintf(&b, "Meaning: %s\n\n", explainProposalSummary(p.Summary))
	}
	fmt.Fprintf(&b, "Status: %s. Type: %s. Risk: %s.\n\n", rec.Status, p.Type, p.RiskLevel)
	if len(p.Actions) > 0 {
		b.WriteString("It would:\n")
		for _, action := range p.Actions {
			fmt.Fprintf(&b, "- %s\n", explainProposalAction(action))
		}
		b.WriteString("\n")
	}
	if len(p.SourceNotes) > 0 {
		b.WriteString("Why Naudia prepared it:\n")
		for _, source := range p.SourceNotes {
			if source.Path == "" && source.Reason == "" {
				continue
			}
			if source.Reason != "" {
				fmt.Fprintf(&b, "- %s: %s\n", source.Path, source.Reason)
			} else {
				fmt.Fprintf(&b, "- %s\n", source.Path)
			}
		}
		b.WriteString("\n")
	}
	if rec.Status == "pending" {
		fmt.Fprintf(&b, "Next: `/show %d` to inspect the patch, `/apply %d` to make the change, or `/reject %d` to discard it.", rec.ID, rec.ID, rec.ID)
	}
	return strings.TrimSpace(b.String()), nil
}

func explainProposalSummary(summary string) string {
	explained := strings.TrimSpace(summary)
	replacements := []struct {
		old string
		new string
	}{
		{
			old: "Create a durable review note sourced from the deterministic vault scan.",
			new: "Naudia wants to write a normal Markdown review note into your vault that records the results of its rule-based vault scan. \"Durable\" means it persists as a note you can read later. \"Deterministic vault scan\" means Naudia found the review data from its parser/index checks, not from model imagination.",
		},
		{
			old: "durable review note",
			new: "persistent Markdown review note",
		},
		{
			old: "deterministic vault scan",
			new: "rule-based vault scan",
		},
	}
	for _, replacement := range replacements {
		explained = strings.ReplaceAll(explained, replacement.old, replacement.new)
	}
	return explained
}

func explainProposalAction(action proposals.ProposalAction) string {
	switch action.Kind {
	case proposals.ActionCreateNote:
		return fmt.Sprintf("create `%s`", action.Path)
	case proposals.ActionAppendToNote:
		return fmt.Sprintf("append content to `%s`", action.Path)
	case proposals.ActionUpdateNote:
		return fmt.Sprintf("replace the content of `%s`", action.Path)
	case proposals.ActionRenameNote:
		return fmt.Sprintf("rename `%s` to `%s`", action.Path, action.NewPath)
	case proposals.ActionMoveNote:
		return fmt.Sprintf("move `%s` to `%s`", action.Path, action.NewPath)
	case proposals.ActionDeleteNote:
		return fmt.Sprintf("delete `%s`", action.Path)
	case proposals.ActionReplaceSection:
		return fmt.Sprintf("replace section `%s` in `%s`", action.Heading, action.Path)
	case proposals.ActionInsertAfterHeading:
		return fmt.Sprintf("insert content after heading `%s` in `%s`", action.Heading, action.Path)
	case proposals.ActionInsertBeforeHeading:
		return fmt.Sprintf("insert content before heading `%s` in `%s`", action.Heading, action.Path)
	case proposals.ActionUpdateTaskStatus:
		return fmt.Sprintf("update a task checkbox in `%s`", action.Path)
	case proposals.ActionAddFrontmatter, proposals.ActionUpdateFrontmatter:
		return fmt.Sprintf("update frontmatter in `%s`", action.Path)
	default:
		return fmt.Sprintf("%s `%s`", action.Kind, action.Path)
	}
}

func updateProposalActionPath(ctx context.Context, r engines.Runner, pm proposals.Manager, id int64, rawPath string) (string, error) {
	rel, err := normalizeVaultRelativePath(r.Config.Vault.Path, rawPath)
	if err != nil {
		return "", err
	}
	p, rec, err := pm.Load(ctx, id)
	if err != nil {
		return "", err
	}
	if rec.Status != "pending" {
		return "", fmt.Errorf("proposal %d is %s, not pending", id, rec.Status)
	}
	indexes, oldPath, err := proposalPathUpdateTargets(p)
	if err != nil {
		return "", err
	}
	for _, idx := range indexes {
		p.Actions[idx].Path = rel
		if p.Actions[idx].Kind == proposals.ActionAppendToNote {
			if hash, err := readVaultFileHash(r.Config.Vault.Path, rel); err == nil {
				p.Actions[idx].ExpectedHash = hash
			} else {
				p.Actions[idx].ExpectedHash = ""
			}
		}
	}
	p.Title = retitleProposalPath(p.Title, oldPath, rel, p.Actions[indexes[0]].Kind)
	if err := pm.Update(ctx, p); err != nil {
		return "", err
	}
	return fmt.Sprintf("Updated proposal %d path from `%s` to `%s`.\n\nReview with `/show %d`, apply with `/apply %d`, or reject with `/reject %d`.", id, oldPath, rel, id, id, id), nil
}

func proposalPathUpdateTargets(p *proposals.Proposal) ([]int, string, error) {
	var indexes []int
	paths := map[string]bool{}
	for i, action := range p.Actions {
		switch action.Kind {
		case proposals.ActionCreateNote, proposals.ActionAppendToNote:
			indexes = append(indexes, i)
			paths[action.Path] = true
		}
	}
	if len(indexes) == 0 {
		return nil, "", fmt.Errorf("proposal %d has no create/append note action path Naudia can safely retarget", p.ID)
	}
	if len(paths) > 1 {
		var parts []string
		for _, idx := range indexes {
			parts = append(parts, fmt.Sprintf("%s=%s", p.Actions[idx].ID, p.Actions[idx].Path))
		}
		return nil, "", fmt.Errorf("proposal %d touches multiple note paths; use /show %d and choose a specific action first: %s", p.ID, p.ID, strings.Join(parts, ", "))
	}
	return indexes, p.Actions[indexes[0]].Path, nil
}

func retitleProposalPath(title, oldPath, newPath string, kind proposals.ActionKind) string {
	if strings.Contains(title, oldPath) {
		return strings.ReplaceAll(title, oldPath, newPath)
	}
	switch kind {
	case proposals.ActionCreateNote:
		return "Create " + newPath
	case proposals.ActionAppendToNote:
		return "Append to " + newPath
	default:
		return title
	}
}

func normalizeVaultRelativePath(vaultPath, raw string) (string, error) {
	cleaned := strings.Trim(strings.TrimSpace(raw), "`\"'")
	cleaned = strings.TrimSuffix(cleaned, ".")
	cleaned = strings.TrimPrefix(cleaned, "file://")
	if unescaped, err := url.PathUnescape(cleaned); err == nil {
		cleaned = unescaped
	}
	cleaned = strings.ReplaceAll(cleaned, "\\", string(filepath.Separator))
	if cleaned == "" {
		return "", fmt.Errorf("path is required")
	}
	rootAbs, err := filepath.Abs(vaultPath)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(cleaned) {
		if rel, ok := relIfInside(rootAbs, cleaned); ok {
			return cleanNoteRelPath(vaultPath, rel)
		}
		if rel, ok := relAfterVaultDir(rootAbs, cleaned); ok {
			return cleanNoteRelPath(vaultPath, rel)
		}
		return "", fmt.Errorf("path must be inside the configured vault: %s", vaultPath)
	}
	if rel, ok := relAfterVaultDir(rootAbs, cleaned); ok {
		cleaned = rel
	}
	return cleanNoteRelPath(vaultPath, cleaned)
}

func relIfInside(rootAbs, path string) (string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func relAfterVaultDir(rootAbs, path string) (string, bool) {
	vaultDir := filepath.Base(rootAbs)
	parts := splitPathParts(path)
	for i, part := range parts {
		if part == vaultDir && i+1 < len(parts) {
			return filepath.Join(parts[i+1:]...), true
		}
	}
	return "", false
}

func splitPathParts(path string) []string {
	path = strings.ReplaceAll(path, "\\", "/")
	var out []string
	for _, part := range strings.Split(path, "/") {
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}

func cleanNoteRelPath(vaultPath, rel string) (string, error) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return "", fmt.Errorf("path escapes vault")
	}
	if filepath.Ext(rel) == "" {
		rel += ".md"
	}
	if _, err := util.ResolveInside(vaultPath, rel); err != nil {
		return "", err
	}
	return rel, nil
}

func readVaultFileHash(vaultPath, rel string) (string, error) {
	full, err := util.ResolveInside(vaultPath, rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return util.SHA256Bytes(data), nil
}
