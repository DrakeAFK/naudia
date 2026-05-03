package vector

import (
	"context"
	"math"
	"sort"

	"github.com/drakeafk/naudia/internal/ai"
	"github.com/drakeafk/naudia/internal/db"
)

type Result struct {
	Chunk db.ChunkRow `json:"chunk"`
	Score float64     `json:"score"`
}

func Search(ctx context.Context, store *db.DB, embedder ai.EmbeddingClient, vaultID int64, model string, query string, limit int, minScore float64) ([]Result, error) {
	if embedder == nil {
		return nil, nil
	}
	embeds, err := embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeds) == 0 || len(embeds[0].Vector) == 0 {
		return nil, nil
	}
	chunks, vectors, err := store.ListEmbeddings(ctx, vaultID, model)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(chunks))
	for i, chunk := range chunks {
		score := Cosine(embeds[0].Vector, vectors[i])
		if score >= minScore {
			results = append(results, Result{Chunk: chunk, Score: score})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func Cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
