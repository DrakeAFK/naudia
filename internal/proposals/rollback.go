package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/DrakeAFK/naudia/internal/util"
)

type RollbackResult struct {
	ProposalID int64      `json:"proposal_id"`
	RolledBack []string   `json:"rolled_back"`
	Conflicts  []Conflict `json:"conflicts"`
	Forced     bool       `json:"forced"`
}

func (m Manager) Rollback(ctx context.Context, proposalID int64, force bool) (RollbackResult, error) {
	rec, err := m.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return RollbackResult{}, err
	}
	if rec.Status != "applied" && rec.Status != "partially_applied" {
		return RollbackResult{}, fmt.Errorf("proposal %d is %s, not applied", proposalID, rec.Status)
	}
	changes, err := m.Store.ListChanges(ctx, proposalID)
	if err != nil {
		return RollbackResult{}, err
	}
	changes, err = m.mergeJournalChanges(ctx, proposalID, changes)
	if err != nil {
		return RollbackResult{}, err
	}
	result := RollbackResult{ProposalID: proposalID, Forced: force}
	for i := len(changes) - 1; i >= 0; i-- {
		change, err := changeFromRecord(changes[i])
		if err != nil {
			return result, err
		}
		if err := m.rollbackChange(ctx, change, force); err != nil {
			conflict := m.conflictFor(change, err)
			result.Conflicts = append(result.Conflicts, conflict)
			if saveErr := m.saveConflict(ctx, conflict); saveErr != nil {
				return result, saveErr
			}
			continue
		}
		result.RolledBack = append(result.RolledBack, change.NotePath)
	}
	if len(result.Conflicts) > 0 {
		_ = m.Store.UpdateProposalStatus(ctx, proposalID, "failed")
		return result, util.Wrap(util.ErrConflict, "rollback produced %d conflict(s)", len(result.Conflicts))
	}
	if err := m.Store.UpdateProposalStatus(ctx, proposalID, "rolled_back"); err != nil {
		return result, err
	}
	return result, nil
}

func (m Manager) mergeJournalChanges(ctx context.Context, proposalID int64, changes []db.ChangeRecord) ([]db.ChangeRecord, error) {
	journals, err := m.Store.ListApplyJournal(ctx, proposalID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, change := range changes {
		seen[change.ActionID+"|"+change.NotePath] = true
	}
	for _, journal := range journals {
		key := journal.ActionID + "|" + journal.NotePath
		if seen[key] {
			continue
		}
		if journal.Status != "applied" && journal.Status != "db_record_failed" {
			continue
		}
		var rec db.ChangeRecord
		if err := json.Unmarshal([]byte(journal.PlannedChangeJSON), &rec); err != nil {
			return nil, err
		}
		rec.ID = -journal.ID
		rec.ProposalID = proposalID
		changes = append(changes, rec)
	}
	return changes, nil
}

func (m Manager) rollbackChange(ctx context.Context, change Change, force bool) error {
	switch ActionKind(change.ActionKind) {
	case ActionCreateNote:
		return m.rollbackCreate(change, force)
	case ActionMoveNote, ActionRenameNote:
		return m.rollbackMove(change, force)
	case ActionDeleteNote:
		return m.rollbackDelete(change, force)
	default:
		return m.rollbackContent(change, force)
	}
}

func (m Manager) rollbackCreate(change Change, force bool) error {
	full, err := util.ResolveInside(m.VaultPath, change.NotePath)
	if err != nil {
		return err
	}
	if !util.FileExists(full) {
		return nil
	}
	current, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	if !force && util.SHA256Bytes(current) != change.AfterHash {
		return util.Wrap(util.ErrConflict, "created file changed after apply: %s", change.NotePath)
	}
	return os.Remove(full)
}

func (m Manager) rollbackMove(change Change, force bool) error {
	if len(change.Anchors) == 0 {
		return util.Wrap(util.ErrConflict, "move metadata is missing")
	}
	srcRel := change.Anchors[0].BeforeContext
	dstRel := change.Anchors[0].AfterContext
	src, err := util.ResolveInside(m.VaultPath, srcRel)
	if err != nil {
		return err
	}
	dst, err := util.ResolveInside(m.VaultPath, dstRel)
	if err != nil {
		return err
	}
	if !util.FileExists(dst) {
		return util.Wrap(util.ErrConflict, "moved destination is missing: %s", dstRel)
	}
	current, err := os.ReadFile(dst)
	if err != nil {
		return err
	}
	if !force && util.SHA256Bytes(current) != change.AfterHash {
		return util.Wrap(util.ErrConflict, "moved file changed after apply: %s", dstRel)
	}
	if util.FileExists(src) && !force {
		return util.Wrap(util.ErrConflict, "original path now exists: %s", srcRel)
	}
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		return err
	}
	if force && util.FileExists(src) {
		if err := os.Remove(src); err != nil {
			return err
		}
	}
	return os.Rename(dst, src)
}

func (m Manager) rollbackDelete(change Change, force bool) error {
	full, err := util.ResolveInside(m.VaultPath, change.NotePath)
	if err != nil {
		return err
	}
	if util.FileExists(full) && !force {
		return util.Wrap(util.ErrConflict, "path now exists after delete: %s", change.NotePath)
	}
	return util.WriteFileAtomic(full, []byte(change.PreviousContent), 0o644)
}

func (m Manager) rollbackContent(change Change, force bool) error {
	full, err := util.ResolveInside(m.VaultPath, change.NotePath)
	if err != nil {
		return err
	}
	currentBytes, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	current := string(currentBytes)
	currentHash := util.SHA256String(current)
	if force || currentHash == change.AfterHash {
		return util.WriteFileAtomic(full, []byte(change.PreviousContent), 0o644)
	}
	if patched, ok := tryStrictInverse(change, current); ok {
		return util.WriteFileAtomic(full, []byte(patched), 0o644)
	}
	if patched, ok := rollbackByAnchors(change, current); ok {
		return util.WriteFileAtomic(full, []byte(patched), 0o644)
	}
	if patched, ok := threeWayRollback(change.PreviousContent, change.AppliedContent, current); ok {
		return util.WriteFileAtomic(full, []byte(patched), 0o644)
	}
	return util.Wrap(util.ErrConflict, "file drift overlaps Naudia change: %s", change.NotePath)
}

func tryStrictInverse(change Change, current string) (string, bool) {
	if len(change.Anchors) == 0 {
		return "", false
	}
	for _, anchor := range change.Anchors {
		if anchor.AfterContext != "" && strings.Count(current, anchor.AfterContext) != 1 {
			return "", false
		}
	}
	patched, ok := ApplyTextPatch(change.InversePatch, current)
	if !ok {
		return "", false
	}
	for _, anchor := range change.Anchors {
		if anchor.BeforeContext != "" && !strings.Contains(patched, anchor.BeforeContext) {
			return "", false
		}
	}
	return patched, true
}

func rollbackByAnchors(change Change, current string) (string, bool) {
	if len(change.Anchors) == 0 {
		return "", false
	}
	result := current
	for _, anchor := range change.Anchors {
		after := anchor.AfterContext
		before := anchor.BeforeContext
		if after == "" {
			continue
		}
		if strings.Count(result, after) != 1 {
			return "", false
		}
		result = strings.Replace(result, after, before, 1)
	}
	return result, true
}

func threeWayRollback(base, applied, current string) (string, bool) {
	if base == applied {
		return current, true
	}
	prefix := commonPrefixLen(base, applied)
	suffix := commonSuffixLen(base[prefix:], applied[prefix:])
	start := lineStart(applied, prefix)
	endApplied := lineEnd(applied, len(applied)-suffix)
	endBase := lineEnd(base, len(base)-suffix)
	if start < prefix {
		prefix = start
	}
	appliedChanged := applied[prefix:endApplied]
	baseChanged := base[prefix:endBase]
	if appliedChanged == "" {
		return "", false
	}
	if strings.Count(current, appliedChanged) != 1 {
		return "", false
	}
	return strings.Replace(current, appliedChanged, baseChanged, 1), true
}

func lineStart(s string, idx int) int {
	if idx > len(s) {
		idx = len(s)
	}
	for idx > 0 && s[idx-1] != '\n' {
		idx--
	}
	return idx
}

func lineEnd(s string, idx int) int {
	if idx > len(s) {
		idx = len(s)
	}
	for idx < len(s) && s[idx] != '\n' {
		idx++
	}
	if idx < len(s) {
		idx++
	}
	return idx
}

func commonPrefixLen(a, b string) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	i := 0
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

func commonSuffixLen(a, b string) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	i := 0
	for i < max && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}

func changeFromRecord(rec db.ChangeRecord) (Change, error) {
	var ranges []AffectedRange
	var anchors []PatchAnchor
	if err := json.Unmarshal([]byte(rec.AffectedRangesJSON), &ranges); err != nil {
		return Change{}, err
	}
	if err := json.Unmarshal([]byte(rec.AnchorsJSON), &anchors); err != nil {
		return Change{}, err
	}
	return Change{
		ID:              rec.ID,
		ProposalID:      rec.ProposalID,
		ActionID:        rec.ActionID,
		NotePath:        rec.NotePath,
		ActionKind:      rec.ActionKind,
		BeforeHash:      rec.BeforeHash,
		AfterHash:       rec.AfterHash,
		PreviousContent: rec.PreviousContent,
		AppliedContent:  rec.AppliedContent,
		ForwardPatch:    rec.ForwardPatch,
		InversePatch:    rec.InversePatch,
		AffectedRanges:  ranges,
		Anchors:         anchors,
	}, nil
}

func (m Manager) conflictFor(change Change, err error) Conflict {
	currentHash := ""
	if full, resolveErr := util.ResolveInside(m.VaultPath, change.NotePath); resolveErr == nil {
		if data, readErr := os.ReadFile(full); readErr == nil {
			currentHash = util.SHA256Bytes(data)
		}
	}
	return Conflict{
		ProposalID:          change.ProposalID,
		ChangeID:            change.ID,
		NotePath:            change.NotePath,
		Reason:              err.Error(),
		OriginalChange:      change.ForwardPatch,
		CurrentFileHash:     currentHash,
		ExpectedAfterHash:   change.AfterHash,
		SuggestedResolution: "Review the conflict artifact, manually remove or restore the affected section, then mark the proposal resolved or rerun rollback with --force if overwriting later edits is acceptable.",
	}
}

func (m Manager) saveConflict(ctx context.Context, conflict Conflict) error {
	data, err := json.MarshalIndent(conflict, "", "  ")
	if err != nil {
		return err
	}
	conflictPath := filepath.Join(m.VaultPath, ".naudia", "conflicts", fmt.Sprintf("proposal-%d.json", conflict.ProposalID))
	if err := util.WriteFileAtomic(conflictPath, data, 0o644); err != nil {
		return err
	}
	_, err = m.Store.SaveConflict(ctx, db.ConflictRecord{
		ProposalID:   conflict.ProposalID,
		ChangeID:     conflict.ChangeID,
		NotePath:     conflict.NotePath,
		Reason:       conflict.Reason,
		ConflictJSON: string(data),
	})
	return err
}
