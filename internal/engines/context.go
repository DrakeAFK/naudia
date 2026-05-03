package engines

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/obsidian"
	"github.com/DrakeAFK/naudia/internal/vector"
)

func (r Runner) BuildContext(ctx context.Context, query string, mode string) (contextpack.Pack, error) {
	budget := contextpack.Budget{
		MaxNotes:               r.Config.Context.MaxNotes,
		MaxChunks:              r.Config.Context.MaxChunks,
		MaxCharsTotal:          r.Config.Context.MaxCharsTotal,
		MaxCharsPerNote:        r.Config.Context.MaxCharsPerNote,
		MaxSemanticMatches:     r.Config.Context.MaxSemanticMatches,
		MinSimilarityThreshold: r.Config.Context.MinSimilarityThreshold,
		IncludeFullNotes:       r.Config.Context.IncludeFullNotes,
		PreferHeadings:         r.Config.Context.PreferHeadings,
	}
	switch mode {
	case "ask":
		budget.MaxNotes = 10
		budget.MaxChunks = 20
		budget.MaxCharsTotal = 30000
	case "project":
		budget.MaxNotes = 12
		budget.MaxChunks = 24
		budget.MaxCharsTotal = 36000
	case "daily":
		budget.MaxNotes = 6
		budget.MaxChunks = 12
		budget.MaxCharsTotal = 18000
	case "links", "structure":
		budget.MaxCharsTotal = 12000
	}
	var candidates []contextpack.Item
	exact, err := r.exactNoteContext(ctx, query)
	if err != nil {
		return contextpack.Pack{}, err
	}
	candidates = append(candidates, exact...)
	linkItems, err := r.linkGraphContext(ctx, query)
	if err != nil {
		return contextpack.Pack{}, err
	}
	candidates = append(candidates, linkItems...)
	tagItems, err := r.tagContext(ctx, query)
	if err != nil {
		return contextpack.Pack{}, err
	}
	candidates = append(candidates, tagItems...)
	if mode == "daily" || mode == "project" || mode == "ask" {
		recentDaily, err := r.recentDailyContext(ctx, query)
		if err != nil {
			return contextpack.Pack{}, err
		}
		candidates = append(candidates, recentDaily...)
	}
	chunks, err := r.Store.SearchChunks(ctx, r.VaultID, query, budget.MaxChunks*2)
	if err != nil {
		return contextpack.Pack{}, err
	}
	queryLower := strings.ToLower(query)
	for _, chunk := range chunks {
		method := "text_search"
		score := 0.55
		reason := "Text search matched the query."
		if strings.EqualFold(chunk.Title, query) {
			method = "exact_title"
			score = 0.95
			reason = "The note title exactly matches the query."
		} else if strings.Contains(strings.ToLower(chunk.NotePath), queryLower) {
			method = "exact_path"
			score = 0.9
			reason = "The note path contains the query."
		}
		candidates = append(candidates, contextpack.Item{
			NotePath:        chunk.NotePath,
			ObsidianURI:     obsidian.BuildOpenNoteURI(r.Config.Vault.Name, chunk.NotePath),
			Heading:         chunk.HeadingContext,
			Excerpt:         chunk.Content,
			RetrievalMethod: method,
			Reason:          reason,
			Score:           score,
		})
	}
	if r.Config.Index.UseEmbeddings && r.AI != nil {
		results, err := vector.Search(ctx, r.Store, r.AI, r.VaultID, r.Config.Ollama.EmbeddingModel, query, r.Config.Context.MaxSemanticMatches, r.Config.Context.MinSimilarityThreshold)
		if err == nil {
			for _, res := range results {
				candidates = append(candidates, contextpack.Item{
					NotePath:        res.Chunk.NotePath,
					ObsidianURI:     obsidian.BuildOpenNoteURI(r.Config.Vault.Name, res.Chunk.NotePath),
					Heading:         res.Chunk.HeadingContext,
					Excerpt:         res.Chunk.Content,
					RetrievalMethod: "semantic",
					Reason:          "Semantic similarity candidate. Treat as supporting evidence only.",
					Score:           res.Score,
				})
			}
		}
	}
	if mode == "project" {
		projectFolder := filepath.Join(r.Config.Vault.Path, "Projects", safeName(query))
		if entries, err := os.ReadDir(projectFolder); err == nil {
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
					continue
				}
				rel := filepath.ToSlash(filepath.Join("Projects", safeName(query), entry.Name()))
				if data, err := os.ReadFile(filepath.Join(projectFolder, entry.Name())); err == nil {
					candidates = append(candidates, contextpack.Item{
						NotePath:        rel,
						ObsidianURI:     obsidian.BuildOpenNoteURI(r.Config.Vault.Name, rel),
						Excerpt:         string(data),
						RetrievalMethod: "folder",
						Reason:          "The note is inside the matching project folder.",
						Score:           0.75,
					})
				}
			}
		}
	}
	return contextpack.Build(candidates, budget), nil
}

func (r Runner) exactNoteContext(ctx context.Context, query string) ([]contextpack.Item, error) {
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM notes n
		JOIN chunks c ON c.note_id = n.id
		WHERE n.vault_id = ?
		  AND (lower(n.title) = lower(?) OR lower(n.path) = lower(?) OR lower(replace(n.path, '.md', '')) = lower(?))
		ORDER BY c.chunk_index
		LIMIT 8
	`, r.VaultID, query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks, err := scanContextChunks(rows)
	if err != nil {
		return nil, err
	}
	return chunksToItems(r.Config.Vault.Name, chunks, "exact_title", "The note title or path exactly matched the query.", 0.95), nil
}

func (r Runner) linkGraphContext(ctx context.Context, query string) ([]contextpack.Item, error) {
	rows, err := r.Store.SQL.QueryContext(ctx, `
		WITH matched AS (
			SELECT id FROM notes
			WHERE vault_id = ?
			  AND (lower(title) LIKE '%' || lower(?) || '%' OR lower(path) LIKE '%' || lower(?) || '%')
			LIMIT 12
		),
		linked AS (
			SELECT l.source_note_id AS note_id, 'backlink' AS method
			FROM links l JOIN matched m ON m.id = l.target_note_id
			UNION
			SELECT l.target_note_id AS note_id, 'outlink' AS method
			FROM links l JOIN matched m ON m.id = l.source_note_id
			WHERE l.target_note_id IS NOT NULL
		)
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0), linked.method
		FROM linked
		JOIN notes n ON n.id = linked.note_id
		JOIN chunks c ON c.note_id = n.id
		WHERE n.vault_id = ?
		ORDER BY linked.method, n.path, c.chunk_index
		LIMIT 16
	`, r.VaultID, query, query, r.VaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []contextpack.Item
	for rows.Next() {
		var c contextChunk
		var method string
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate, &method); err != nil {
			return nil, err
		}
		score := 0.78
		reason := "This note is linked from a note matching the query."
		if method == "backlink" {
			score = 0.8
			reason = "This note links to a note matching the query."
		}
		items = append(items, chunkToItem(r.Config.Vault.Name, c, method, reason, score))
	}
	return items, rows.Err()
}

func (r Runner) tagContext(ctx context.Context, query string) ([]contextpack.Item, error) {
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}
	var items []contextpack.Item
	for _, token := range tokens {
		rows, err := r.Store.SQL.QueryContext(ctx, `
			SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
			FROM tags t
			JOIN notes n ON n.id = t.note_id
			JOIN chunks c ON c.note_id = n.id
			WHERE n.vault_id = ? AND lower(t.tag) = lower(?)
			ORDER BY n.path, c.chunk_index
			LIMIT 8
		`, r.VaultID, strings.TrimPrefix(token, "#"))
		if err != nil {
			return nil, err
		}
		chunks, err := scanContextChunks(rows)
		closeErr := rows.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		items = append(items, chunksToItems(r.Config.Vault.Name, chunks, "tag", "The note has a tag matching the query.", 0.68)...)
	}
	return items, nil
}

func (r Runner) recentDailyContext(ctx context.Context, query string) ([]contextpack.Item, error) {
	cutoff := time.Now().AddDate(0, 0, -r.Config.Context.RecentDailyNoteDays).Format("2006-01-02")
	dailyPrefix := strings.Trim(r.Config.Daily.Folder, "/") + "/"
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM notes n
		JOIN chunks c ON c.note_id = n.id
		WHERE n.vault_id = ? AND n.path LIKE ? AND n.path >= ? AND lower(c.content) LIKE '%' || lower(?) || '%'
		ORDER BY n.path DESC, c.chunk_index
		LIMIT 10
	`, r.VaultID, dailyPrefix+"%", dailyPrefix+cutoff, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks, err := scanContextChunks(rows)
	if err != nil {
		return nil, err
	}
	return chunksToItems(r.Config.Vault.Name, chunks, "recent_daily", "A recent daily note mentions the query.", 0.6), nil
}

type contextChunk struct {
	ID             int64
	NoteID         int64
	NotePath       string
	Title          string
	ChunkIndex     int
	Content        string
	ContentHash    string
	HeadingContext string
	TokenEstimate  int
}

func scanContextChunks(rows *sql.Rows) ([]contextChunk, error) {
	var chunks []contextChunk
	for rows.Next() {
		var c contextChunk
		if err := rows.Scan(&c.ID, &c.NoteID, &c.NotePath, &c.Title, &c.ChunkIndex, &c.Content, &c.ContentHash, &c.HeadingContext, &c.TokenEstimate); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func chunksToItems(vaultName string, chunks []contextChunk, method, reason string, score float64) []contextpack.Item {
	items := make([]contextpack.Item, 0, len(chunks))
	for _, chunk := range chunks {
		items = append(items, chunkToItem(vaultName, chunk, method, reason, score))
	}
	return items
}

func chunkToItem(vaultName string, chunk contextChunk, method, reason string, score float64) contextpack.Item {
	return contextpack.Item{
		NotePath:        chunk.NotePath,
		ObsidianURI:     obsidian.BuildOpenNoteURI(vaultName, chunk.NotePath),
		Heading:         chunk.HeadingContext,
		Excerpt:         chunk.Content,
		RetrievalMethod: method,
		Reason:          reason,
		Score:           score,
	}
}

func queryTokens(query string) []string {
	fields := strings.Fields(strings.ToLower(query))
	var out []string
	for _, field := range fields {
		field = strings.Trim(field, "#.,:;!?()[]{}\"'")
		if len(field) >= 3 {
			out = append(out, field)
		}
	}
	return out
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	if s == "" {
		return "Untitled"
	}
	return s
}
