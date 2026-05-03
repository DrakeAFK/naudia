package contextpack

import "testing"

func TestBuildPrefersExactOverSemanticAndEnforcesBudget(t *testing.T) {
	items := []Item{
		{NotePath: "Semantic.md", Excerpt: "semantic", RetrievalMethod: "semantic", Score: 0.99},
		{NotePath: "Exact.md", Excerpt: "exact", RetrievalMethod: "exact_title", Score: 0.5},
		{NotePath: "Weak.md", Excerpt: "weak", RetrievalMethod: "semantic", Score: 0.2},
	}
	pack := Build(items, Budget{MaxNotes: 2, MaxChunks: 2, MaxCharsTotal: 100, MaxCharsPerNote: 100, MaxSemanticMatches: 1, MinSimilarityThreshold: 0.68})
	if len(pack.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(pack.Items))
	}
	if pack.Items[0].NotePath != "Exact.md" {
		t.Fatalf("first item = %s, want Exact.md", pack.Items[0].NotePath)
	}
	for _, item := range pack.Items {
		if item.NotePath == "Weak.md" {
			t.Fatalf("weak semantic item should have been dropped")
		}
	}
}
