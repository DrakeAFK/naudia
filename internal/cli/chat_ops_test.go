package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DrakeAFK/naudia/internal/config"
	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/DrakeAFK/naudia/internal/engines"
	"github.com/DrakeAFK/naudia/internal/proposals"
	"github.com/DrakeAFK/naudia/internal/util"
)

func TestRunNaudiaOperationListsPendingProposalsWithoutVaultContext(t *testing.T) {
	ctx := context.Background()
	r, pm, cleanup := testChatOpsRunner(t, ctx)
	defer cleanup()
	if _, err := pm.Save(ctx, &proposals.Proposal{
		Type:      proposals.TypeNoteCreate,
		Title:     "Create inbox note",
		Summary:   "Create a note from chat.",
		Actions:   []proposals.ProposalAction{{ID: "create", Kind: proposals.ActionCreateNote, Path: "Inbox.md", Content: "# Inbox\n"}},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	result, handled, err := runNaudiaOperation(ctx, r, pm, "What are my current proposals from naudia?")
	if err != nil {
		t.Fatalf("runNaudiaOperation() error = %v", err)
	}
	if !handled {
		t.Fatal("expected proposal question to be handled before vault assistant")
	}
	if !strings.Contains(result.Answer, "Pending Naudia proposals") || !strings.Contains(result.Answer, "Create inbox note") {
		t.Fatalf("answer did not list proposals:\n%s", result.Answer)
	}
	if strings.Contains(result.Answer, "Source") || len(result.Context.Items) != 0 {
		t.Fatalf("proposal operation should not include vault context: %#v\n%s", result.Context, result.Answer)
	}
}

func TestRunNaudiaOperationUpdatesPendingProposalPath(t *testing.T) {
	ctx := context.Background()
	r, pm, cleanup := testChatOpsRunner(t, ctx)
	defer cleanup()
	id, err := pm.Save(ctx, &proposals.Proposal{
		Type:      proposals.TypeNoteCreate,
		Title:     "Create obsidian/DrakeAFK/wrong.md",
		Summary:   "Create a note from chat.",
		Actions:   []proposals.ProposalAction{{ID: "create", Kind: proposals.ActionCreateNote, Path: "obsidian/DrakeAFK/wrong.md", Content: "# Wrong\n"}},
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	result, handled, err := runNaudiaOperation(ctx, r, pm, "update proposal 1 file path to be /obsidian/DrakeAFK/cmdsetgo.md")
	if err != nil {
		t.Fatalf("runNaudiaOperation() error = %v", err)
	}
	if !handled {
		t.Fatal("expected proposal update to be handled before vault assistant")
	}
	if !strings.Contains(result.Answer, "cmdsetgo.md") {
		t.Fatalf("answer did not confirm updated path:\n%s", result.Answer)
	}
	updated, _, err := pm.Load(ctx, id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := updated.Actions[0].Path; got != "cmdsetgo.md" {
		t.Fatalf("proposal path = %q, want cmdsetgo.md", got)
	}
	if !strings.Contains(updated.Title, "cmdsetgo.md") {
		t.Fatalf("proposal title was not retitled: %q", updated.Title)
	}
}

func TestNormalizeVaultRelativePathStripsVaultPrefix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "DrakeAFK")
	if err := util.EnsureNaudiaDirs(root); err != nil {
		t.Fatalf("EnsureNaudiaDirs() error = %v", err)
	}
	got, err := normalizeVaultRelativePath(root, "/obsidian/DrakeAFK/cmdsetgo.md")
	if err != nil {
		t.Fatalf("normalizeVaultRelativePath() error = %v", err)
	}
	if got != "cmdsetgo.md" {
		t.Fatalf("path = %q, want cmdsetgo.md", got)
	}
}

func testChatOpsRunner(t *testing.T, ctx context.Context) (engines.Runner, proposals.Manager, func()) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "DrakeAFK")
	if err := util.EnsureNaudiaDirs(root); err != nil {
		t.Fatalf("EnsureNaudiaDirs() error = %v", err)
	}
	store, err := db.Open(ctx, filepath.Join(root, ".naudia", "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	cfg := config.Default()
	cfg.Vault.Path = root
	cfg.Vault.Name = "DrakeAFK"
	vaultID, err := store.UpsertVault(ctx, cfg.Vault.Name, cfg.Vault.Path, false, true)
	if err != nil {
		_ = store.Close()
		t.Fatalf("UpsertVault() error = %v", err)
	}
	r := engines.Runner{Config: cfg, Store: store, VaultID: vaultID}
	pm := proposals.Manager{VaultPath: root, VaultID: vaultID, Store: store}
	return r, pm, func() { _ = store.Close() }
}
