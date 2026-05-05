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

var proposalPathRequestRE = regexp.MustCompile(`(?i)\b(?:update|change|set)\s+proposal\s+([0-9]+)\b.*\b(?:file\s+path|path)\b.*\bto(?:\s+be)?\s+(.+)$`)

func runNaudiaOperation(ctx context.Context, r engines.Runner, pm proposals.Manager, message string) (engines.AssistResult, bool, error) {
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)
	if isProposalListRequest(lower) {
		answer, err := pendingProposalAnswer(ctx, pm)
		return engines.AssistResult{Answer: answer, Context: contextpack.Pack{}}, true, err
	}
	if id, ok := parseNaturalShowProposal(lower); ok {
		p, rec, err := pm.Load(ctx, id)
		if err != nil {
			return engines.AssistResult{}, true, err
		}
		return engines.AssistResult{Answer: proposals.RenderDetails(*p, rec), Context: contextpack.Pack{}}, true, nil
	}
	if match := proposalPathRequestRE.FindStringSubmatch(trimmed); len(match) == 3 {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return engines.AssistResult{}, true, err
		}
		answer, err := updateProposalActionPath(ctx, r, pm, id, match[2])
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
