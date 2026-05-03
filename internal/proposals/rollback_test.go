package proposals

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/drakeafk/naudia/internal/db"
	"github.com/drakeafk/naudia/internal/util"
)

func TestRollbackPreservesUnrelatedManualEdit(t *testing.T) {
	ctx := context.Background()
	root, manager := testManager(t, ctx)
	path := filepath.Join(root, "Note.md")
	original := "# Note\n\n## Target\nold\n\n## Other\nkeep\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &Proposal{
		Type:      TypeNoteUpdate,
		Title:     "Replace section",
		Summary:   "Replace target section.",
		RiskLevel: RiskLow,
		CreatedAt: time.Now().UTC(),
		Actions: []ProposalAction{{
			ID:           "replace-target",
			Kind:         ActionReplaceSection,
			Path:         "Note.md",
			Heading:      "Target",
			Content:      "## Target\nnew\n",
			ExpectedHash: util.SHA256String(original),
		}},
	}
	id, err := manager.Save(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(ctx, id); err != nil {
		t.Fatal(err)
	}
	drifted := "# Note\n\n## Target\nnew\n\n## Other\nmanual edit\n"
	if err := os.WriteFile(path, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Rollback(ctx, id, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "## Target\nold") {
		t.Fatalf("target section was not rolled back:\n%s", got)
	}
	if !strings.Contains(got, "## Other\nmanual edit") {
		t.Fatalf("manual edit was not preserved:\n%s", got)
	}
}

func TestRollbackSameSectionManualEditCreatesConflict(t *testing.T) {
	ctx := context.Background()
	root, manager := testManager(t, ctx)
	path := filepath.Join(root, "Note.md")
	original := "# Note\n\n## Target\nold\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &Proposal{
		Type:      TypeNoteUpdate,
		Title:     "Replace section",
		Summary:   "Replace target section.",
		RiskLevel: RiskLow,
		CreatedAt: time.Now().UTC(),
		Actions: []ProposalAction{{
			ID:           "replace-target",
			Kind:         ActionReplaceSection,
			Path:         "Note.md",
			Heading:      "Target",
			Content:      "## Target\nnew\n",
			ExpectedHash: util.SHA256String(original),
		}},
	}
	id, err := manager.Save(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Note\n\n## Target\nnew plus manual edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Rollback(ctx, id, false); err == nil {
		t.Fatal("rollback succeeded despite same-section drift")
	}
	if _, err := os.Stat(filepath.Join(root, ".naudia", "conflicts", "proposal-1.json")); err != nil {
		t.Fatalf("conflict artifact missing: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "new plus manual edit") {
		t.Fatalf("conflicted file was overwritten:\n%s", string(data))
	}
}

func testManager(t *testing.T, ctx context.Context) (string, Manager) {
	t.Helper()
	root := t.TempDir()
	if err := util.EnsureNaudiaDirs(root); err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(ctx, filepath.Join(root, ".naudia", "naudia.sqlite"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	vaultID, err := store.UpsertVault(ctx, "Test", root, false, true)
	if err != nil {
		t.Fatal(err)
	}
	return root, Manager{VaultPath: root, VaultID: vaultID, Store: store}
}
