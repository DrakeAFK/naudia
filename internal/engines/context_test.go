package engines

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/DrakeAFK/naudia/internal/config"
	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/DrakeAFK/naudia/internal/vault"
)

func TestBuildContextFindsRootNoteFromNaturalQuestion(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := db.Open(ctx, filepath.Join(root, ".naudia", "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	cfg := config.Default()
	cfg.Vault.Path = root
	cfg.Vault.Name = "Test"
	cfg.Index.UseEmbeddings = false
	vaultID, err := store.UpsertVault(ctx, cfg.Vault.Name, cfg.Vault.Path, false, true)
	if err != nil {
		t.Fatalf("UpsertVault() error = %v", err)
	}
	note := vault.ParseNote("Local AI.md", filepath.Join(root, "Local AI.md"), "# Local AI\n\nLocal AI keeps private notes on the user's machine.\n")
	related := vault.ParseNote("Projects/Test App.md", filepath.Join(root, "Projects", "Test App.md"), "# Test App\n\nThe Test App uses Ollama for local AI.\n")
	if _, err := store.IndexScan(ctx, vaultID, vault.ScanResult{VaultPath: root, Notes: []vault.Note{note, related}}, cfg.Index.ChunkSize, cfg.Index.ChunkOverlap, true); err != nil {
		t.Fatalf("IndexScan() error = %v", err)
	}

	r := Runner{Config: cfg, Store: store, VaultID: vaultID}
	pack, err := r.BuildContext(ctx, "what is local AI?", "ask")
	if err != nil {
		t.Fatalf("BuildContext() error = %v", err)
	}
	if len(pack.Items) == 0 {
		t.Fatal("expected context for natural question")
	}
	if pack.Items[0].NotePath != "Local AI.md" {
		t.Fatalf("first note = %s, want Local AI.md", pack.Items[0].NotePath)
	}
	if len(pack.Items) != 1 {
		t.Fatalf("items = %d, want focused context with one source", len(pack.Items))
	}
}
