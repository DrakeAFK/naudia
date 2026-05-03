package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/drakeafk/naudia/internal/db"
	"github.com/drakeafk/naudia/internal/util"
)

type Manager struct {
	VaultPath string
	VaultID   int64
	Store     *db.DB
}

type ApplyResult struct {
	ProposalID int64    `json:"proposal_id"`
	Applied    []string `json:"applied"`
	Failed     []string `json:"failed"`
}

func (m Manager) Save(ctx context.Context, p *Proposal) (int64, error) {
	if err := Validate(p); err != nil {
		return 0, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return 0, err
	}
	patch := RenderPatchPreview(m.VaultPath, p)
	id, err := m.Store.SaveProposal(ctx, m.VaultID, string(p.Type), p.Title, p.Summary, string(data), patch)
	if err != nil {
		return 0, err
	}
	p.ID = id
	fileData, _ := json.MarshalIndent(p, "", "  ")
	path := filepath.Join(m.VaultPath, ".naudia", "proposals", fmt.Sprintf("proposal-%d.json", id))
	_ = util.WriteFileAtomic(path, fileData, 0o644)
	return id, nil
}

func (m Manager) Load(ctx context.Context, id int64) (*Proposal, db.ProposalRecord, error) {
	rec, err := m.Store.GetProposal(ctx, id)
	if err != nil {
		return nil, rec, err
	}
	var p Proposal
	if err := json.Unmarshal([]byte(rec.ProposalJSON), &p); err != nil {
		return nil, rec, err
	}
	p.ID = rec.ID
	return &p, rec, nil
}

func (m Manager) Apply(ctx context.Context, proposalID int64) (ApplyResult, error) {
	p, rec, err := m.Load(ctx, proposalID)
	if err != nil {
		return ApplyResult{}, err
	}
	if rec.Status != "pending" {
		return ApplyResult{}, fmt.Errorf("proposal %d is %s, not pending", proposalID, rec.Status)
	}
	result := ApplyResult{ProposalID: proposalID}
	for _, action := range p.Actions {
		change, journalID, err := m.applyAction(ctx, action, proposalID)
		if err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", action.ID, err))
			_ = m.Store.UpdateProposalStatus(ctx, proposalID, "partially_applied")
			if len(result.Applied) == 0 {
				_ = m.Store.UpdateProposalStatus(ctx, proposalID, "failed")
			}
			return result, err
		}
		rangesJSON, _ := json.Marshal(change.AffectedRanges)
		anchorsJSON, _ := json.Marshal(change.Anchors)
		id, err := m.Store.SaveChange(ctx, db.ChangeRecord{
			ProposalID:         proposalID,
			ActionID:           change.ActionID,
			NotePath:           change.NotePath,
			ActionKind:         change.ActionKind,
			BeforeHash:         change.BeforeHash,
			AfterHash:          change.AfterHash,
			PreviousContent:    change.PreviousContent,
			AppliedContent:     change.AppliedContent,
			ForwardPatch:       change.ForwardPatch,
			InversePatch:       change.InversePatch,
			AffectedRangesJSON: string(rangesJSON),
			AnchorsJSON:        string(anchorsJSON),
		})
		if err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", action.ID, err))
			_ = m.Store.UpdateProposalStatus(ctx, proposalID, "partially_applied")
			if journalID > 0 {
				_ = m.Store.CompleteApplyJournal(ctx, journalID, "db_record_failed", err.Error())
			}
			return result, err
		}
		if journalID > 0 {
			_ = m.Store.CompleteApplyJournal(ctx, journalID, "applied", "")
		}
		result.Applied = append(result.Applied, fmt.Sprintf("%d:%s", id, action.Path))
	}
	if err := m.Store.UpdateProposalStatus(ctx, proposalID, "applied"); err != nil {
		return result, err
	}
	return result, nil
}

func (m Manager) applyAction(ctx context.Context, action ProposalAction, proposalID int64) (Change, int64, error) {
	if action.ID == "" {
		action.ID = string(action.Kind) + "-" + action.Path
	}
	full, err := util.ResolveInside(m.VaultPath, action.Path)
	if err != nil {
		return Change{}, 0, err
	}
	if action.Kind == ActionMoveNote || action.Kind == ActionRenameNote {
		return m.applyMove(ctx, action, proposalID, full)
	}
	if action.Kind == ActionDeleteNote {
		return m.applyDelete(ctx, action, proposalID, full)
	}
	exists := util.FileExists(full)
	var previous string
	if exists {
		b, err := os.ReadFile(full)
		if err != nil {
			return Change{}, 0, err
		}
		previous = string(b)
	}
	if action.ExpectedHash != "" && util.SHA256String(previous) != action.ExpectedHash {
		return Change{}, 0, util.Wrap(util.ErrStale, "proposal action %s is stale for %s", action.ID, action.Path)
	}
	applied, anchors, ranges, err := applyContentAction(action, previous, exists)
	if err != nil {
		return Change{}, 0, err
	}
	afterHash := util.SHA256String(applied)
	change := Change{
		ProposalID:      proposalID,
		ActionID:        action.ID,
		NotePath:        action.Path,
		ActionKind:      string(action.Kind),
		BeforeHash:      util.SHA256String(previous),
		AfterHash:       afterHash,
		PreviousContent: previous,
		AppliedContent:  applied,
		ForwardPatch:    TextPatch(previous, applied),
		InversePatch:    TextPatch(applied, previous),
		AffectedRanges:  ranges,
		Anchors:         anchors,
		AppliedAt:       time.Now().UTC(),
	}
	journalID, err := m.journalPlannedChange(ctx, change)
	if err != nil {
		return Change{}, 0, err
	}
	if err := util.WriteFileAtomic(full, []byte(applied), 0o644); err != nil {
		_ = m.Store.CompleteApplyJournal(ctx, journalID, "write_failed", err.Error())
		return Change{}, journalID, err
	}
	verified, err := os.ReadFile(full)
	if err != nil {
		_ = m.Store.CompleteApplyJournal(ctx, journalID, "verify_failed", err.Error())
		return Change{}, journalID, err
	}
	if util.SHA256Bytes(verified) != afterHash {
		err := fmt.Errorf("post-write hash verification failed for %s", action.Path)
		_ = m.Store.CompleteApplyJournal(ctx, journalID, "verify_failed", err.Error())
		return Change{}, journalID, err
	}
	return change, journalID, nil
}

func (m Manager) applyMove(ctx context.Context, action ProposalAction, proposalID int64, src string) (Change, int64, error) {
	dst, err := util.ResolveInside(m.VaultPath, action.NewPath)
	if err != nil {
		return Change{}, 0, err
	}
	previousBytes, err := os.ReadFile(src)
	if err != nil {
		return Change{}, 0, err
	}
	previous := string(previousBytes)
	if action.ExpectedHash != "" && util.SHA256String(previous) != action.ExpectedHash {
		return Change{}, 0, util.Wrap(util.ErrStale, "proposal action %s is stale for %s", action.ID, action.Path)
	}
	if util.FileExists(dst) && !action.AllowOverwrite {
		return Change{}, 0, util.Wrap(util.ErrConflict, "destination exists: %s", action.NewPath)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return Change{}, 0, err
	}
	change := Change{
		ProposalID:      proposalID,
		ActionID:        action.ID,
		NotePath:        action.Path,
		ActionKind:      string(action.Kind),
		BeforeHash:      util.SHA256String(previous),
		AfterHash:       util.SHA256String(previous),
		PreviousContent: previous,
		AppliedContent:  previous,
		ForwardPatch:    fmt.Sprintf("move %s -> %s", action.Path, action.NewPath),
		InversePatch:    fmt.Sprintf("move %s -> %s", action.NewPath, action.Path),
		Anchors: []PatchAnchor{{
			BeforeContext: action.Path,
			AfterContext:  action.NewPath,
			SectionID:     "move",
		}},
		AppliedAt: time.Now().UTC(),
	}
	journalID, err := m.journalPlannedChange(ctx, change)
	if err != nil {
		return Change{}, 0, err
	}
	if err := os.Rename(src, dst); err != nil {
		_ = m.Store.CompleteApplyJournal(ctx, journalID, "write_failed", err.Error())
		return Change{}, journalID, err
	}
	return change, journalID, nil
}

func (m Manager) applyDelete(ctx context.Context, action ProposalAction, proposalID int64, full string) (Change, int64, error) {
	previousBytes, err := os.ReadFile(full)
	if err != nil {
		return Change{}, 0, err
	}
	previous := string(previousBytes)
	if action.ExpectedHash != "" && util.SHA256String(previous) != action.ExpectedHash {
		return Change{}, 0, util.Wrap(util.ErrStale, "proposal action %s is stale for %s", action.ID, action.Path)
	}
	change := Change{
		ProposalID:      proposalID,
		ActionID:        action.ID,
		NotePath:        action.Path,
		ActionKind:      string(action.Kind),
		BeforeHash:      util.SHA256String(previous),
		AfterHash:       "",
		PreviousContent: previous,
		AppliedContent:  "",
		ForwardPatch:    TextPatch(previous, ""),
		InversePatch:    TextPatch("", previous),
		Anchors: []PatchAnchor{{
			BeforeContext: previous,
			AfterContext:  "",
			SectionID:     "delete",
		}},
		AppliedAt: time.Now().UTC(),
	}
	journalID, err := m.journalPlannedChange(ctx, change)
	if err != nil {
		return Change{}, 0, err
	}
	if err := os.Remove(full); err != nil {
		_ = m.Store.CompleteApplyJournal(ctx, journalID, "write_failed", err.Error())
		return Change{}, journalID, err
	}
	return change, journalID, nil
}

func (m Manager) journalPlannedChange(ctx context.Context, change Change) (int64, error) {
	rangesJSON, _ := json.Marshal(change.AffectedRanges)
	anchorsJSON, _ := json.Marshal(change.Anchors)
	rec := db.ChangeRecord{
		ProposalID:         change.ProposalID,
		ActionID:           change.ActionID,
		NotePath:           change.NotePath,
		ActionKind:         change.ActionKind,
		BeforeHash:         change.BeforeHash,
		AfterHash:          change.AfterHash,
		PreviousContent:    change.PreviousContent,
		AppliedContent:     change.AppliedContent,
		ForwardPatch:       change.ForwardPatch,
		InversePatch:       change.InversePatch,
		AffectedRangesJSON: string(rangesJSON),
		AnchorsJSON:        string(anchorsJSON),
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return 0, err
	}
	path := filepath.Join(m.VaultPath, ".naudia", "changes", fmt.Sprintf("proposal-%d-%s.json", change.ProposalID, sanitizeActionID(change.ActionID)))
	if err := util.WriteFileAtomic(path, data, 0o644); err != nil {
		return 0, err
	}
	return m.Store.SaveApplyJournal(ctx, rec, string(data))
}

func sanitizeActionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "action"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(id)
}

func applyContentAction(action ProposalAction, previous string, exists bool) (string, []PatchAnchor, []AffectedRange, error) {
	switch action.Kind {
	case ActionCreateNote:
		if exists && !action.AllowOverwrite {
			return "", nil, nil, util.Wrap(util.ErrConflict, "file already exists: %s", action.Path)
		}
		content := ensureFinalNewline(action.Content)
		return content, []PatchAnchor{{BeforeContext: "", AfterContext: content, Heading: action.Heading, SectionID: action.ID}}, affected(previous, content), nil
	case ActionUpdateNote:
		content := ensureFinalNewline(action.Content)
		return content, []PatchAnchor{{BeforeContext: previous, AfterContext: content, Heading: action.Heading, SectionID: action.ID}}, affected(previous, content), nil
	case ActionAppendToNote:
		insert := ensureLeadingAndFinalNewline(action.Content, previous)
		content := previous + insert
		return content, []PatchAnchor{{BeforeContext: "", AfterContext: insert, Heading: action.Heading, SectionID: action.ID}}, affected("", insert), nil
	case ActionReplaceSection:
		content, before, after, err := replaceSection(previous, action.Heading, action.HeadingLevel, ensureFinalNewline(action.Content))
		if err != nil {
			return "", nil, nil, err
		}
		return content, []PatchAnchor{{BeforeContext: before, AfterContext: after, Heading: action.Heading, SectionID: action.ID}}, affected(before, after), nil
	case ActionInsertAfterHeading:
		content, inserted, err := insertAroundHeading(previous, action.Heading, action.HeadingLevel, ensureFinalNewline(action.Content), true)
		if err != nil {
			return "", nil, nil, err
		}
		return content, []PatchAnchor{{BeforeContext: "", AfterContext: inserted, Heading: action.Heading, SectionID: action.ID}}, affected("", inserted), nil
	case ActionInsertBeforeHeading:
		content, inserted, err := insertAroundHeading(previous, action.Heading, action.HeadingLevel, ensureFinalNewline(action.Content), false)
		if err != nil {
			return "", nil, nil, err
		}
		return content, []PatchAnchor{{BeforeContext: "", AfterContext: inserted, Heading: action.Heading, SectionID: action.ID}}, affected("", inserted), nil
	case ActionUpdateLines:
		content, anchors, err := updateLines(previous, action.LineEdits)
		if err != nil {
			return "", nil, nil, err
		}
		return content, anchors, rangesFromAnchors(anchors), nil
	case ActionRemoveLines:
		content, removed, err := removeLines(previous, action.StartLine, action.EndLine)
		if err != nil {
			return "", nil, nil, err
		}
		return content, []PatchAnchor{{BeforeContext: removed, AfterContext: "", SectionID: action.ID}}, affected(removed, ""), nil
	case ActionUpdateTaskStatus:
		content, anchors, err := updateTaskStatus(previous, action)
		if err != nil {
			return "", nil, nil, err
		}
		return content, anchors, rangesFromAnchors(anchors), nil
	case ActionAddFrontmatter, ActionUpdateFrontmatter:
		content, before, after := updateFrontmatter(previous, action.Frontmatter)
		return content, []PatchAnchor{{BeforeContext: before, AfterContext: after, SectionID: action.ID}}, affected(before, after), nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported action kind: %s", action.Kind)
	}
}

func replaceSection(content, heading string, level int, replacement string) (string, string, string, error) {
	lines := splitKeep(content)
	start, end, err := sectionRange(lines, heading, level)
	if err != nil {
		return "", "", "", err
	}
	before := strings.Join(lines[start:end], "")
	if replacement == "" {
		replacement = "\n"
	}
	lines = append(lines[:start], append([]string{replacement}, lines[end:]...)...)
	return strings.Join(lines, ""), before, replacement, nil
}

func insertAroundHeading(content, heading string, level int, insertion string, after bool) (string, string, error) {
	lines := splitKeep(content)
	start, _, err := sectionRange(lines, heading, level)
	if err != nil {
		return "", "", err
	}
	pos := start
	if after {
		pos = start + 1
	}
	lines = append(lines[:pos], append([]string{insertion}, lines[pos:]...)...)
	return strings.Join(lines, ""), insertion, nil
}

func sectionRange(lines []string, heading string, level int) (int, int, error) {
	heading = strings.TrimSpace(heading)
	matches := []int{}
	for i, line := range lines {
		hashes := countHeadingHashes(line)
		if hashes == 0 {
			continue
		}
		if level > 0 && hashes != level {
			continue
		}
		text := strings.TrimSpace(strings.Trim(strings.TrimSpace(line[hashes:]), "#"))
		if strings.EqualFold(text, heading) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return 0, 0, util.Wrap(util.ErrNotFound, "heading not found: %s", heading)
	}
	if len(matches) > 1 {
		return 0, 0, util.Wrap(util.ErrConflict, "heading is ambiguous: %s", heading)
	}
	start := matches[0]
	startLevel := countHeadingHashes(lines[start])
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if h := countHeadingHashes(lines[i]); h > 0 && h <= startLevel {
			end = i
			break
		}
	}
	return start, end, nil
}

func updateLines(content string, edits []LineEdit) (string, []PatchAnchor, error) {
	if len(edits) == 0 {
		return content, nil, errorsNew("line_edits is empty")
	}
	lines := splitKeep(content)
	sorted := append([]LineEdit(nil), edits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LineNumber > sorted[j].LineNumber })
	anchors := make([]PatchAnchor, 0, len(sorted))
	for _, edit := range sorted {
		if edit.LineNumber <= 0 || edit.LineNumber > len(lines) {
			return "", nil, fmt.Errorf("line %d out of range", edit.LineNumber)
		}
		idx := edit.LineNumber - 1
		old := strings.TrimRight(lines[idx], "\n")
		if old != edit.OldText {
			return "", nil, util.Wrap(util.ErrStale, "line %d changed", edit.LineNumber)
		}
		newLine := edit.NewText
		if strings.HasSuffix(lines[idx], "\n") {
			newLine += "\n"
		}
		lines[idx] = newLine
		anchors = append(anchors, PatchAnchor{BeforeContext: lineWithOriginalNewline(edit.OldText, newLine), AfterContext: newLine, SectionID: fmt.Sprintf("line-%d", edit.LineNumber)})
	}
	return strings.Join(lines, ""), anchors, nil
}

func removeLines(content string, start, end int) (string, string, error) {
	lines := splitKeep(content)
	if start <= 0 || end < start || end > len(lines) {
		return "", "", fmt.Errorf("invalid line range %d-%d", start, end)
	}
	removed := strings.Join(lines[start-1:end], "")
	lines = append(lines[:start-1], lines[end:]...)
	return strings.Join(lines, ""), removed, nil
}

func updateTaskStatus(content string, action ProposalAction) (string, []PatchAnchor, error) {
	lines := splitKeep(content)
	if action.StartLine <= 0 || action.StartLine > len(lines) {
		return "", nil, fmt.Errorf("task line %d out of range", action.StartLine)
	}
	if action.TaskCompleted == nil {
		return "", nil, errorsNew("task_completed is required")
	}
	idx := action.StartLine - 1
	old := lines[idx]
	newLine := old
	if *action.TaskCompleted {
		newLine = strings.Replace(newLine, "[ ]", "[x]", 1)
	} else {
		newLine = strings.Replace(strings.Replace(newLine, "[x]", "[ ]", 1), "[X]", "[ ]", 1)
	}
	if old == newLine {
		return "", nil, util.Wrap(util.ErrStale, "task line %d did not contain the expected checkbox", action.StartLine)
	}
	lines[idx] = newLine
	return strings.Join(lines, ""), []PatchAnchor{{BeforeContext: old, AfterContext: newLine, SectionID: fmt.Sprintf("task-%d", action.StartLine)}}, nil
}

func updateFrontmatter(content string, updates map[string]string) (string, string, string) {
	if len(updates) == 0 {
		return content, "", ""
	}
	lines := splitKeep(content)
	start, end := -1, -1
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		start = 0
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
	}
	if start == -1 || end == -1 {
		keys := sortedKeys(updates)
		var b strings.Builder
		b.WriteString("---\n")
		for _, key := range keys {
			fmt.Fprintf(&b, "%s: %s\n", key, updates[key])
		}
		b.WriteString("---\n")
		after := b.String()
		return after + content, "", after
	}
	before := strings.Join(lines[start:end+1], "")
	existing := map[string]int{}
	for i := start + 1; i < end; i++ {
		if parts := strings.SplitN(lines[i], ":", 2); len(parts) == 2 {
			existing[strings.TrimSpace(parts[0])] = i
		}
	}
	for _, key := range sortedKeys(updates) {
		line := fmt.Sprintf("%s: %s\n", key, updates[key])
		if idx, ok := existing[key]; ok {
			lines[idx] = line
		} else {
			lines = append(lines[:end], append([]string{line}, lines[end:]...)...)
			end++
		}
	}
	after := strings.Join(lines[start:end+1], "")
	return strings.Join(lines, ""), before, after
}

func RenderPatchPreview(vaultPath string, p *Proposal) string {
	var b strings.Builder
	for _, action := range p.Actions {
		full, err := util.ResolveInside(vaultPath, action.Path)
		if err != nil {
			continue
		}
		var previous string
		if data, err := os.ReadFile(full); err == nil {
			previous = string(data)
		}
		if action.Kind == ActionMoveNote || action.Kind == ActionRenameNote {
			fmt.Fprintf(&b, "move %s -> %s\n", action.Path, action.NewPath)
			continue
		}
		if action.Kind == ActionDeleteNote {
			b.WriteString(UnifiedPatch(action.Path, previous, ""))
			b.WriteString("\n")
			continue
		}
		applied, _, _, err := applyContentAction(action, previous, util.FileExists(full))
		if err != nil {
			fmt.Fprintf(&b, "# %s: %v\n", action.Path, err)
			continue
		}
		b.WriteString(UnifiedPatch(action.Path, previous, applied))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func affected(before, after string) []AffectedRange {
	return []AffectedRange{{
		StartLineBefore: 1,
		EndLineBefore:   countLines(before),
		StartLineAfter:  1,
		EndLineAfter:    countLines(after),
		BeforeTextHash:  util.SHA256String(before),
		AfterTextHash:   util.SHA256String(after),
	}}
}

func rangesFromAnchors(anchors []PatchAnchor) []AffectedRange {
	out := make([]AffectedRange, 0, len(anchors))
	for _, a := range anchors {
		out = append(out, affected(a.BeforeContext, a.AfterContext)...)
	}
	return out
}

func splitKeep(content string) []string {
	if content == "" {
		return []string{}
	}
	parts := strings.SplitAfter(content, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func ensureFinalNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func ensureLeadingAndFinalNewline(insert, previous string) string {
	insert = ensureFinalNewline(insert)
	if previous == "" || strings.HasSuffix(previous, "\n") {
		return insert
	}
	return "\n" + insert
}

func countHeadingHashes(line string) int {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i == len(line) || line[i] != ' ' {
		return 0
	}
	if i > 6 {
		return 0
	}
	return i
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func lineWithOriginalNewline(old, newLine string) string {
	if strings.HasSuffix(newLine, "\n") {
		return old + "\n"
	}
	return old
}

func errorsNew(s string) error {
	return fmt.Errorf("%s", s)
}
