package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drakeafk/naudia/internal/util"
	"github.com/drakeafk/naudia/internal/vault"
)

func (d *DB) UpsertVault(ctx context.Context, name, path string, cliEnabled, uriEnabled bool) (int64, error) {
	now := util.NowText()
	_, err := d.SQL.ExecContext(ctx, `
		INSERT INTO vaults(name, path, obsidian_cli_enabled, obsidian_uri_enabled, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
		  name = excluded.name,
		  obsidian_cli_enabled = excluded.obsidian_cli_enabled,
		  obsidian_uri_enabled = excluded.obsidian_uri_enabled,
		  updated_at = excluded.updated_at
	`, name, path, boolInt(cliEnabled), boolInt(uriEnabled), now, now)
	if err != nil {
		return 0, err
	}
	var id int64
	err = d.SQL.QueryRowContext(ctx, `SELECT id FROM vaults WHERE path = ?`, path).Scan(&id)
	return id, err
}

func (d *DB) VaultID(ctx context.Context, vaultPath string) (int64, error) {
	var id int64
	err := d.SQL.QueryRowContext(ctx, `SELECT id FROM vaults WHERE path = ?`, vaultPath).Scan(&id)
	return id, err
}

func (d *DB) Status(ctx context.Context, vaultPath string) (Status, error) {
	var st Status
	st.DatabasePath = d.Path
	st.VectorAvailable = d.VectorAvailable
	err := d.SQL.QueryRowContext(ctx, `SELECT id, name, path FROM vaults WHERE path = ?`, vaultPath).Scan(&st.VaultID, &st.VaultName, &st.VaultPath)
	if err != nil {
		return st, err
	}
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes WHERE vault_id = ?`, st.VaultID).Scan(&st.Notes)
	_ = d.SQL.QueryRowContext(ctx, `SELECT COALESCE(MAX(indexed_at), '') FROM notes WHERE vault_id = ?`, st.VaultID).Scan(&st.LastScan)
	_ = d.SQL.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM embeddings e
		JOIN notes n ON n.id = e.note_id
		WHERE n.vault_id = ?
	`, st.VaultID).Scan(&st.EmbeddingsStored)
	if st.VectorAvailable {
		_ = d.SQL.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM vector_chunks vc
			JOIN notes n ON n.id = vc.note_id
			WHERE n.vault_id = ?
		`, st.VaultID).Scan(&st.VectorIndexed)
	}
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM proposals WHERE vault_id = ? AND status = 'pending'`, st.VaultID).Scan(&st.PendingProposals)
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM proposals WHERE vault_id = ? AND status = 'applied'`, st.VaultID).Scan(&st.AppliedProposals)
	return st, nil
}

func (d *DB) IndexScan(ctx context.Context, vaultID int64, result vault.ScanResult, chunkSize, overlap int, force bool) (ScanStats, error) {
	stats := ScanStats{}
	seen := map[string]bool{}
	err := d.Tx(ctx, func(tx *sql.Tx) error {
		for _, note := range result.Notes {
			seen[note.Path] = true
			var existingHash string
			var existingID int64
			err := tx.QueryRowContext(ctx, `SELECT id, content_hash FROM notes WHERE vault_id = ? AND path = ?`, vaultID, note.Path).Scan(&existingID, &existingHash)
			if err == nil && existingHash == note.ContentHash && !force {
				stats.NotesSkipped++
				continue
			}
			aliasesJSON, _ := json.Marshal(note.Aliases)
			now := util.NowText()
			created := note.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
			modified := note.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
			_, err = tx.ExecContext(ctx, `
				INSERT INTO notes(vault_id, path, title, aliases_json, frontmatter_json, content_hash, word_count, created_at, modified_at, indexed_at)
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(vault_id, path) DO UPDATE SET
				  title = excluded.title,
				  aliases_json = excluded.aliases_json,
				  frontmatter_json = excluded.frontmatter_json,
				  content_hash = excluded.content_hash,
				  word_count = excluded.word_count,
				  modified_at = excluded.modified_at,
				  indexed_at = excluded.indexed_at
			`, vaultID, note.Path, note.Title, string(aliasesJSON), vault.ParseFrontmatterJSON(note.Frontmatter), note.ContentHash, note.WordCount, created, modified, now)
			if err != nil {
				return err
			}
			var noteID int64
			if err := tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE vault_id = ? AND path = ?`, vaultID, note.Path).Scan(&noteID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM headings WHERE note_id = ?`, noteID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE source_note_id = ?`, noteID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE note_id = ?`, noteID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE note_id = ?`, noteID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE note_id = ?`, noteID); err != nil {
				return err
			}
			for _, h := range note.Headings {
				if _, err := tx.ExecContext(ctx, `INSERT INTO headings(note_id, level, text, line_number) VALUES(?, ?, ?, ?)`, noteID, h.Level, h.Text, h.LineNumber); err != nil {
					return err
				}
			}
			for _, l := range note.Links {
				var targetID any
				if l.TargetPath != "" {
					var id int64
					if err := tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE vault_id = ? AND path = ?`, vaultID, l.TargetPath).Scan(&id); err == nil {
						targetID = id
					}
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO links(source_note_id, target_raw, target_heading, target_note_id, link_text, line_number, resolved) VALUES(?, ?, ?, ?, ?, ?, ?)`,
					noteID, l.TargetRaw, l.TargetHeading, targetID, l.LinkText, l.LineNumber, boolInt(l.Resolved)); err != nil {
					return err
				}
			}
			for _, tag := range note.Tags {
				if _, err := tx.ExecContext(ctx, `INSERT INTO tags(note_id, tag) VALUES(?, ?)`, noteID, tag.Name); err != nil {
					return err
				}
			}
			for _, task := range note.Tasks {
				if _, err := tx.ExecContext(ctx, `INSERT INTO tasks(note_id, text, completed, line_number, inferred, project_guess, due_date_guess, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
					noteID, task.Text, boolInt(task.Completed), task.LineNumber, boolInt(task.Inferred), task.ProjectGuess, task.DueDateGuess, now); err != nil {
					return err
				}
			}
			for _, chunk := range vault.ChunkNote(note, chunkSize, overlap) {
				if _, err := tx.ExecContext(ctx, `INSERT INTO chunks(note_id, chunk_index, content, content_hash, heading_context, token_estimate, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`,
					noteID, chunk.ChunkIndex, chunk.Content, chunk.ContentHash, chunk.HeadingContext, chunk.TokenEstimate, now); err != nil {
					return err
				}
				stats.ChunksIndexed++
			}
			stats.NotesIndexed++
		}
		rows, err := tx.QueryContext(ctx, `SELECT path FROM notes WHERE vault_id = ?`, vaultID)
		if err != nil {
			return err
		}
		defer rows.Close()
		var missing []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				return err
			}
			if !seen[p] {
				missing = append(missing, p)
			}
		}
		for _, p := range missing {
			if _, err := tx.ExecContext(ctx, `DELETE FROM notes WHERE vault_id = ? AND path = ?`, vaultID, p); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	return stats, err
}

func (d *DB) ListNotes(ctx context.Context, vaultID int64) ([]NoteRow, error) {
	rows, err := d.SQL.QueryContext(ctx, `SELECT id, path, title, aliases_json, content_hash, word_count, COALESCE(modified_at, ''), indexed_at FROM notes WHERE vault_id = ? ORDER BY path`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NoteRow
	for rows.Next() {
		var n NoteRow
		if err := rows.Scan(&n.ID, &n.Path, &n.Title, &n.AliasesJSON, &n.ContentHash, &n.WordCount, &n.ModifiedAt, &n.IndexedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (d *DB) NoteByPath(ctx context.Context, vaultID int64, path string) (NoteRow, error) {
	var n NoteRow
	err := d.SQL.QueryRowContext(ctx, `SELECT id, path, title, aliases_json, content_hash, word_count, COALESCE(modified_at, ''), indexed_at FROM notes WHERE vault_id = ? AND path = ?`, vaultID, path).
		Scan(&n.ID, &n.Path, &n.Title, &n.AliasesJSON, &n.ContentHash, &n.WordCount, &n.ModifiedAt, &n.IndexedAt)
	return n, err
}

func (d *DB) ListChunks(ctx context.Context, vaultID int64) ([]ChunkRow, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM chunks c
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ?
		ORDER BY n.path, c.chunk_index
	`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChunkRow
	for rows.Next() {
		var c ChunkRow
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) SearchChunks(ctx context.Context, vaultID int64, query string, limit int) ([]ChunkRow, error) {
	if limit <= 0 {
		limit = 20
	}
	like := "%" + strings.ToLower(query) + "%"
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM chunks c
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ?
		  AND (lower(n.path) LIKE ? OR lower(n.title) LIKE ? OR lower(c.content) LIKE ? OR lower(COALESCE(c.heading_context, '')) LIKE ?)
		ORDER BY
		  CASE WHEN lower(n.title) = lower(?) THEN 0
		       WHEN lower(n.path) LIKE ? THEN 1
		       ELSE 2 END,
		  length(c.content) ASC
		LIMIT ?
	`, vaultID, like, like, like, like, query, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChunkRow
	for rows.Next() {
		var c ChunkRow
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) StoreEmbedding(ctx context.Context, chunk ChunkRow, model string, vector []float64) error {
	data, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	now := util.NowText()
	_, err = d.SQL.ExecContext(ctx, `
		INSERT INTO embeddings(note_id, chunk_id, model, content_hash, dimensions, embedding_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chunk_id, model) DO UPDATE SET
		  content_hash = excluded.content_hash,
		  dimensions = excluded.dimensions,
		  embedding_json = excluded.embedding_json,
		  created_at = excluded.created_at
	`, chunk.NoteID, chunk.ID, model, chunk.ContentHash, len(vector), string(data), now)
	if err != nil {
		return err
	}
	if d.VectorAvailable {
		_, _ = d.SQL.ExecContext(ctx, `INSERT OR REPLACE INTO vector_chunks(rowid, chunk_id, note_id, model, content_hash) VALUES(?, ?, ?, ?, ?)`, chunk.ID, chunk.ID, chunk.NoteID, model, chunk.ContentHash)
		embeddingJSON, _ := json.Marshal(vector)
		_, _ = d.SQL.ExecContext(ctx, `INSERT OR REPLACE INTO vec_chunks(rowid, embedding) VALUES(?, ?)`, chunk.ID, string(embeddingJSON))
	}
	return nil
}

func (d *DB) BackfillVectorIndex(ctx context.Context) (int, error) {
	if !d.VectorAvailable {
		return 0, nil
	}
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT e.chunk_id, e.note_id, e.model, e.content_hash, e.dimensions, e.embedding_json
		FROM embeddings e
		JOIN chunks c ON c.id = e.chunk_id
		LEFT JOIN vector_chunks vc
		  ON vc.rowid = e.chunk_id
		 AND vc.model = e.model
		 AND vc.content_hash = e.content_hash
		WHERE e.embedding_json IS NOT NULL
		  AND e.content_hash = c.content_hash
		  AND vc.rowid IS NULL
	`)
	if err != nil {
		return 0, err
	}

	type vectorBackfillRow struct {
		chunkID     int64
		noteID      int64
		model       string
		contentHash string
		vectorJSON  string
	}
	var pending []vectorBackfillRow
	for rows.Next() {
		var chunkID, noteID int64
		var model, contentHash, raw string
		var dimensions int
		if err := rows.Scan(&chunkID, &noteID, &model, &contentHash, &dimensions, &raw); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if dimensions != 768 {
			continue
		}
		var vector []float64
		if err := json.Unmarshal([]byte(raw), &vector); err != nil || len(vector) != 768 {
			continue
		}
		embeddingJSON, _ := json.Marshal(vector)
		pending = append(pending, vectorBackfillRow{
			chunkID:     chunkID,
			noteID:      noteID,
			model:       model,
			contentHash: contentHash,
			vectorJSON:  string(embeddingJSON),
		})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	indexed := 0
	for _, row := range pending {
		if _, err := d.SQL.ExecContext(ctx, `INSERT OR REPLACE INTO vector_chunks(rowid, chunk_id, note_id, model, content_hash) VALUES(?, ?, ?, ?, ?)`, row.chunkID, row.chunkID, row.noteID, row.model, row.contentHash); err != nil {
			return indexed, err
		}
		if _, err := d.SQL.ExecContext(ctx, `INSERT OR REPLACE INTO vec_chunks(rowid, embedding) VALUES(?, ?)`, row.chunkID, row.vectorJSON); err != nil {
			return indexed, err
		}
		indexed++
	}
	return indexed, nil
}

func (d *DB) ChunksNeedingEmbeddings(ctx context.Context, vaultID int64, model string, limit int) ([]ChunkRow, error) {
	if limit <= 0 {
		limit = 128
	}
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM chunks c
		JOIN notes n ON n.id = c.note_id
		LEFT JOIN embeddings e ON e.chunk_id = c.id AND e.model = ?
		WHERE n.vault_id = ? AND (e.id IS NULL OR e.content_hash != c.content_hash)
		ORDER BY n.path, c.chunk_index
		LIMIT ?
	`, model, vaultID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChunkRow
	for rows.Next() {
		var c ChunkRow
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) ListEmbeddings(ctx context.Context, vaultID int64, model string) ([]ChunkRow, [][]float64, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0), e.embedding_json
		FROM embeddings e
		JOIN chunks c ON c.id = e.chunk_id
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ? AND e.model = ? AND e.embedding_json IS NOT NULL
	`, vaultID, model)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var chunks []ChunkRow
	var vectors [][]float64
	for rows.Next() {
		var c ChunkRow
		var raw string
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate, &raw); err != nil {
			return nil, nil, err
		}
		var vec []float64
		if err := json.Unmarshal([]byte(raw), &vec); err != nil || len(vec) == 0 {
			continue
		}
		chunks = append(chunks, c)
		vectors = append(vectors, vec)
	}
	return chunks, vectors, rows.Err()
}

type VectorSearchRow struct {
	Chunk ChunkRow
	Score float64
}

func (d *DB) SearchVecChunks(ctx context.Context, vaultID int64, model string, queryJSON string, limit int) ([]VectorSearchRow, error) {
	if !d.VectorAvailable {
		return nil, fmt.Errorf("sqlite-vec unavailable")
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash,
		       COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0), distance
		FROM vec_chunks v
		JOIN vector_chunks vc ON vc.rowid = v.rowid
		JOIN chunks c ON c.id = vc.chunk_id
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ? AND vc.model = ? AND v.embedding MATCH ? AND k = ?
		ORDER BY distance
	`, vaultID, model, queryJSON, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VectorSearchRow
	for rows.Next() {
		var row VectorSearchRow
		var distance float64
		if err := rows.Scan(&row.Chunk.ID, &row.Chunk.NoteID, &row.Chunk.NotePath, &row.Chunk.Title, &row.Chunk.ChunkIndex, &row.Chunk.Content, &row.Chunk.ContentHash, &row.Chunk.HeadingContext, &row.Chunk.TokenEstimate, &distance); err != nil {
			return nil, err
		}
		row.Score = 1 / (1 + distance)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d *DB) SaveProposal(ctx context.Context, vaultID int64, typ, title, summary, proposalJSON, patchText string) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, `
		INSERT INTO proposals(vault_id, type, title, summary, status, proposal_json, patch_text, created_at)
		VALUES(?, ?, ?, ?, 'pending', ?, ?, ?)
	`, vaultID, typ, title, summary, proposalJSON, patchText, util.NowText())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetProposal(ctx context.Context, id int64) (ProposalRecord, error) {
	var p ProposalRecord
	err := d.SQL.QueryRowContext(ctx, `
		SELECT id, vault_id, type, title, summary, status, proposal_json, COALESCE(patch_text, ''), created_at,
		       COALESCE(applied_at, ''), COALESCE(rejected_at, ''), COALESCE(rolled_back_at, '')
		FROM proposals WHERE id = ?
	`, id).Scan(&p.ID, &p.VaultID, &p.Type, &p.Title, &p.Summary, &p.Status, &p.ProposalJSON, &p.PatchText, &p.CreatedAt, &p.AppliedAt, &p.RejectedAt, &p.RolledBackAt)
	return p, err
}

func (d *DB) ListProposals(ctx context.Context, vaultID int64, status string, all bool) ([]ProposalRecord, error) {
	query := `
		SELECT id, vault_id, type, title, summary, status, proposal_json, COALESCE(patch_text, ''), created_at,
		       COALESCE(applied_at, ''), COALESCE(rejected_at, ''), COALESCE(rolled_back_at, '')
		FROM proposals WHERE vault_id = ?`
	args := []any{vaultID}
	if !all && status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id DESC`
	rows, err := d.SQL.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProposalRecord
	for rows.Next() {
		var p ProposalRecord
		if err := rows.Scan(&p.ID, &p.VaultID, &p.Type, &p.Title, &p.Summary, &p.Status, &p.ProposalJSON, &p.PatchText, &p.CreatedAt, &p.AppliedAt, &p.RejectedAt, &p.RolledBackAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) UpdateProposalStatus(ctx context.Context, id int64, status string) error {
	field := ""
	switch status {
	case "applied", "partially_applied":
		field = ", applied_at = datetime('now')"
	case "rejected":
		field = ", rejected_at = datetime('now')"
	case "rolled_back":
		field = ", rolled_back_at = datetime('now')"
	}
	_, err := d.SQL.ExecContext(ctx, fmt.Sprintf(`UPDATE proposals SET status = ?%s WHERE id = ?`, field), status, id)
	return err
}

func (d *DB) SaveChange(ctx context.Context, c ChangeRecord) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, `
		INSERT INTO changes(proposal_id, action_id, note_path, action_kind, before_hash, after_hash, previous_content, applied_content, forward_patch, inverse_patch, affected_ranges_json, anchors_json, applied_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ProposalID, c.ActionID, c.NotePath, c.ActionKind, c.BeforeHash, c.AfterHash, c.PreviousContent, c.AppliedContent, c.ForwardPatch, c.InversePatch, c.AffectedRangesJSON, c.AnchorsJSON, util.NowText())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) SaveApplyJournal(ctx context.Context, c ChangeRecord, plannedJSON string) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, `
		INSERT INTO apply_journal(proposal_id, action_id, note_path, action_kind, planned_change_json, status, created_at)
		VALUES(?, ?, ?, ?, ?, 'planned', ?)
	`, c.ProposalID, c.ActionID, c.NotePath, c.ActionKind, plannedJSON, util.NowText())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) CompleteApplyJournal(ctx context.Context, id int64, status string, errorText string) error {
	_, err := d.SQL.ExecContext(ctx, `UPDATE apply_journal SET status = ?, completed_at = datetime('now'), error = ? WHERE id = ?`, status, errorText, id)
	return err
}

func (d *DB) ListApplyJournal(ctx context.Context, proposalID int64) ([]ApplyJournalRecord, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT id, proposal_id, action_id, note_path, action_kind, planned_change_json, status, created_at,
		       COALESCE(completed_at, ''), COALESCE(error, '')
		FROM apply_journal
		WHERE proposal_id = ?
		ORDER BY id ASC
	`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ApplyJournalRecord
	for rows.Next() {
		var rec ApplyJournalRecord
		if err := rows.Scan(&rec.ID, &rec.ProposalID, &rec.ActionID, &rec.NotePath, &rec.ActionKind, &rec.PlannedChangeJSON, &rec.Status, &rec.CreatedAt, &rec.CompletedAt, &rec.Error); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (d *DB) ListChanges(ctx context.Context, proposalID int64) ([]ChangeRecord, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT id, proposal_id, action_id, note_path, action_kind, COALESCE(before_hash, ''), COALESCE(after_hash, ''),
		       COALESCE(previous_content, ''), COALESCE(applied_content, ''), COALESCE(forward_patch, ''), COALESCE(inverse_patch, ''),
		       COALESCE(affected_ranges_json, '[]'), COALESCE(anchors_json, '[]'), applied_at
		FROM changes WHERE proposal_id = ? ORDER BY id ASC
	`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChangeRecord
	for rows.Next() {
		var c ChangeRecord
		if err := rows.Scan(&c.ID, &c.ProposalID, &c.ActionID, &c.NotePath, &c.ActionKind, &c.BeforeHash, &c.AfterHash, &c.PreviousContent, &c.AppliedContent, &c.ForwardPatch, &c.InversePatch, &c.AffectedRangesJSON, &c.AnchorsJSON, &c.AppliedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) SaveConflict(ctx context.Context, c ConflictRecord) (int64, error) {
	res, err := d.SQL.ExecContext(ctx, `
		INSERT INTO rollback_conflicts(proposal_id, change_id, note_path, reason, conflict_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, c.ProposalID, c.ChangeID, c.NotePath, c.Reason, c.ConflictJSON, util.NowText())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
