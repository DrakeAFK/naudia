package db

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestOpenEnablesBundledSQLiteVec(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if !store.VectorAvailable {
		t.Fatal("expected bundled sqlite-vec to be available after migrations")
	}
}

func TestSearchVecChunksUsesBundledSQLiteVec(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	vaultID, err := store.UpsertVault(ctx, "Test", "/tmp/test-vault", false, true)
	if err != nil {
		t.Fatalf("UpsertVault() error = %v", err)
	}
	noteRes, err := store.SQL.ExecContext(ctx, `
		INSERT INTO notes(vault_id, path, title, aliases_json, content_hash, indexed_at)
		VALUES(?, 'Search.md', 'Search', '[]', 'note-hash', '2026-05-03T00:00:00Z')
	`, vaultID)
	if err != nil {
		t.Fatalf("insert note error = %v", err)
	}
	noteID, err := noteRes.LastInsertId()
	if err != nil {
		t.Fatalf("note LastInsertId() error = %v", err)
	}
	chunkRes, err := store.SQL.ExecContext(ctx, `
		INSERT INTO chunks(note_id, chunk_index, content, content_hash, heading_context, token_estimate, created_at)
		VALUES(?, 0, 'local-first semantic search', 'chunk-hash', '', 4, '2026-05-03T00:00:00Z')
	`, noteID)
	if err != nil {
		t.Fatalf("insert chunk error = %v", err)
	}
	chunkID, err := chunkRes.LastInsertId()
	if err != nil {
		t.Fatalf("chunk LastInsertId() error = %v", err)
	}

	vector := make([]float64, 768)
	vector[0] = 1
	chunk := ChunkRow{
		ID:          chunkID,
		NoteID:      noteID,
		NotePath:    "Search.md",
		Title:       "Search",
		ChunkIndex:  0,
		Content:     "local-first semantic search",
		ContentHash: "chunk-hash",
	}
	if err := store.StoreEmbedding(ctx, chunk, "test-embed", vector); err != nil {
		t.Fatalf("StoreEmbedding() error = %v", err)
	}
	queryJSON, err := json.Marshal(vector)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rows, err := store.SearchVecChunks(ctx, vaultID, "test-embed", string(queryJSON), 3)
	if err != nil {
		t.Fatalf("SearchVecChunks() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("SearchVecChunks() returned %d rows, want 1", len(rows))
	}
	if rows[0].Chunk.ID != chunkID {
		t.Fatalf("SearchVecChunks() returned chunk %d, want %d", rows[0].Chunk.ID, chunkID)
	}
}
