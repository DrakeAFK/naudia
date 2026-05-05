package engines

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

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
	tokenItems, err := r.tokenContext(ctx, query, budget.MaxChunks*3)
	if err != nil {
		return contextpack.Pack{}, err
	}
	candidates = append(candidates, tokenItems...)
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
	if mode == "ask" {
		candidates = focusHighConfidenceAskContext(candidates)
	}
	pack := contextpack.Build(candidates, budget)
	if mode == "ask" && len(pack.Items) == 0 {
		fallback, err := r.smallVaultContext(ctx, budget.MaxChunks)
		if err != nil {
			return contextpack.Pack{}, err
		}
		pack = contextpack.Build(fallback, budget)
	}
	return pack, nil
}

func focusHighConfidenceAskContext(items []contextpack.Item) []contextpack.Item {
	focus := map[string]bool{}
	for _, item := range items {
		switch item.RetrievalMethod {
		case "exact_title", "exact_path", "title_token", "path_token":
			if item.Score >= 0.86 {
				focus[item.NotePath] = true
			}
		}
	}
	if len(focus) == 0 {
		return items
	}
	out := make([]contextpack.Item, 0, len(items))
	for _, item := range items {
		if focus[item.NotePath] {
			out = append(out, item)
			continue
		}
		switch item.RetrievalMethod {
		case "backlink", "outlink", "tag":
			out = append(out, item)
		}
	}
	return out
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

func (r Runner) tokenContext(ctx context.Context, query string, limit int) ([]contextpack.Item, error) {
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 24
	}
	if len(tokens) > 8 {
		tokens = tokens[:8]
	}
	where := make([]string, 0, len(tokens))
	args := []any{r.VaultID}
	for _, token := range tokens {
		like := "%" + token + "%"
		where = append(where, `(lower(n.path) LIKE ? OR lower(n.title) LIKE ? OR lower(COALESCE(n.aliases_json, '')) LIKE ? OR lower(COALESCE(c.heading_context, '')) LIKE ? OR lower(c.content) LIKE ?)`)
		args = append(args, like, like, like, like, like)
	}
	args = append(args, limit)
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM chunks c
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ? AND (`+strings.Join(where, " OR ")+`)
		ORDER BY
		  CASE
		    WHEN lower(n.title) LIKE '%' || lower(?) || '%' THEN 0
		    WHEN lower(n.path) LIKE '%' || lower(?) || '%' THEN 1
		    ELSE 2
		  END,
		  n.path,
		  c.chunk_index
		LIMIT ?
	`, append(args[:len(args)-1], strings.Join(tokens, " "), strings.Join(tokens, " "), limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks, err := scanContextChunks(rows)
	if err != nil {
		return nil, err
	}
	items := make([]contextpack.Item, 0, len(chunks))
	phrase := strings.Join(tokens, " ")
	for _, chunk := range chunks {
		method, reason, score := scoreTokenChunk(chunk, tokens, phrase)
		if method == "" {
			continue
		}
		items = append(items, chunkToItem(r.Config.Vault.Name, chunk, method, reason, score))
	}
	return items, nil
}

func scoreTokenChunk(chunk contextChunk, tokens []string, phrase string) (string, string, float64) {
	title := strings.ToLower(chunk.Title)
	path := strings.ToLower(strings.TrimSuffix(chunk.NotePath, ".md"))
	heading := strings.ToLower(chunk.HeadingContext)
	content := strings.ToLower(chunk.Content)
	if phrase != "" && (strings.Contains(title, phrase) || strings.Contains(path, phrase)) {
		return "title_token", "The note title or path contains the key phrase from the question.", 0.9
	}
	titleHits := countContains(title, tokens)
	pathHits := countContains(path, tokens)
	headingHits := countContains(heading, tokens)
	contentHits := countContains(content, tokens)
	all := len(tokens)
	switch {
	case all > 0 && titleHits == all:
		return "title_token", "The note title contains all key terms from the question.", 0.88
	case all > 0 && pathHits == all:
		return "path_token", "The note path contains all key terms from the question.", 0.86
	case titleHits > 0 || pathHits > 0:
		return "title_token", "The note title or path contains key terms from the question.", 0.78 + 0.02*float64(titleHits+pathHits)
	case all > 0 && headingHits == all:
		return "heading_token", "A heading contains all key terms from the question.", 0.74
	case headingHits > 0:
		return "heading_token", "A heading contains key terms from the question.", 0.68 + 0.02*float64(headingHits)
	case all > 0 && contentHits == all:
		return "token_search", "The note content contains all key terms from the question.", 0.66
	case contentHits > 0:
		return "token_search", "The note content contains key terms from the question.", 0.56 + 0.02*float64(contentHits)
	default:
		return "", "", 0
	}
}

func countContains(haystack string, needles []string) int {
	count := 0
	for _, needle := range needles {
		if containsTerm(haystack, needle) {
			count++
		}
	}
	return count
}

func containsTerm(haystack string, needle string) bool {
	if len(needle) > 2 {
		return strings.Contains(haystack, needle)
	}
	for _, field := range strings.FieldsFunc(haystack, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		if field == needle {
			return true
		}
	}
	return false
}

func (r Runner) smallVaultContext(ctx context.Context, limit int) ([]contextpack.Item, error) {
	var notes int
	if err := r.Store.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes WHERE vault_id = ?`, r.VaultID).Scan(&notes); err != nil {
		return nil, err
	}
	if notes == 0 || notes > 12 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.Store.SQL.QueryContext(ctx, `
		SELECT c.id, c.note_id, n.path, n.title, c.chunk_index, c.content, c.content_hash, COALESCE(c.heading_context, ''), COALESCE(c.token_estimate, 0)
		FROM chunks c
		JOIN notes n ON n.id = c.note_id
		WHERE n.vault_id = ?
		ORDER BY
		  CASE WHEN instr(n.path, '/') = 0 THEN 0 ELSE 1 END,
		  n.path,
		  c.chunk_index
		LIMIT ?
	`, r.VaultID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks, err := scanContextChunks(rows)
	if err != nil {
		return nil, err
	}
	return chunksToItems(r.Config.Vault.Name, chunks, "small_vault", "The vault is small enough to include broad context after no direct match.", 0.5), nil
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
	query = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, query)
	fields := strings.Fields(query)
	seen := map[string]bool{}
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || queryStopwords[field] {
			continue
		}
		if len(field) < 2 {
			continue
		}
		if seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	return out
}

var queryStopwords = map[string]bool{
	"a": true, "about": true, "after": true, "all": true, "also": true, "am": true, "an": true,
	"and": true, "any": true, "are": true, "as": true, "at": true, "be": true, "been": true,
	"but": true, "by": true, "can": true, "could": true, "did": true, "do": true, "does": true,
	"explain": true, "for": true, "from": true, "had": true, "has": true, "have": true, "how": true,
	"i": true, "in": true, "into": true, "is": true, "it": true, "its": true, "me": true,
	"my": true, "note": true, "notes": true, "of": true, "on": true, "or": true, "our": true,
	"please": true, "show": true, "tell": true, "that": true, "the": true, "their": true,
	"these": true, "this": true, "those": true, "to": true, "was": true, "we": true, "were": true,
	"what": true, "when": true, "where": true, "which": true, "who": true, "why": true, "with": true,
	"would": true, "you": true, "your": true,
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
