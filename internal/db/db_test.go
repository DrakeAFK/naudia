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

func TestMigrateRepairsMissingSQLiteVecTable(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.SQL.ExecContext(ctx, `DROP TABLE IF EXISTS vec_chunks`); err != nil {
		t.Fatalf("drop vec_chunks error = %v", err)
	}
	store.VectorAvailable = false

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if !store.VectorAvailable {
		t.Fatal("expected migration repair to recreate bundled sqlite-vec table")
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

func TestBackfillVectorIndexCopiesExistingEmbeddings(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "naudia.sqlite"), nil)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	vaultID, err := store.UpsertVault(ctx, "Test", "/tmp/backfill-vault", false, true)
	if err != nil {
		t.Fatalf("UpsertVault() error = %v", err)
	}
	noteRes, err := store.SQL.ExecContext(ctx, `
		INSERT INTO notes(vault_id, path, title, aliases_json, content_hash, indexed_at)
		VALUES(?, 'Backfill.md', 'Backfill', '[]', 'note-hash', '2026-05-03T00:00:00Z')
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
		VALUES(?, 0, 'existing embedding', 'chunk-hash', '', 2, '2026-05-03T00:00:00Z')
	`, noteID)
	if err != nil {
		t.Fatalf("insert chunk error = %v", err)
	}
	chunkID, err := chunkRes.LastInsertId()
	if err != nil {
		t.Fatalf("chunk LastInsertId() error = %v", err)
	}

	vector := make([]float64, 768)
	vector[2] = 1
	raw, err := json.Marshal(vector)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if _, err := store.SQL.ExecContext(ctx, `
		INSERT INTO embeddings(note_id, chunk_id, model, content_hash, dimensions, embedding_json, created_at)
		VALUES(?, ?, 'test-embed', 'chunk-hash', 768, ?, '2026-05-03T00:00:00Z')
	`, noteID, chunkID, string(raw)); err != nil {
		t.Fatalf("insert embedding error = %v", err)
	}

	indexed, err := store.BackfillVectorIndex(ctx)
	if err != nil {
		t.Fatalf("BackfillVectorIndex() error = %v", err)
	}
	if indexed != 1 {
		t.Fatalf("BackfillVectorIndex() indexed %d rows, want 1", indexed)
	}
	status, err := store.Status(ctx, "/tmp/backfill-vault")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.VectorIndexed != 1 {
		t.Fatalf("Status().VectorIndexed = %d, want 1", status.VectorIndexed)
	}
}
