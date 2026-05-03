package engines

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/drakeafk/naudia/internal/contextpack"
	"github.com/drakeafk/naudia/internal/obsidian"
	"github.com/drakeafk/naudia/internal/vector"
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

func safeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	if s == "" {
		return "Untitled"
	}
	return s
}
