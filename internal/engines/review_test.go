package engines

import "testing"

func TestDuplicateIssueDetectsAIRestatement(t *testing.T) {
	existing := []Issue{{
		Category:        "Links",
		Severity:        "low",
		Description:     "89 notes have no incoming or outgoing resolved links.",
		SuggestedAction: "Run naudia links to prepare conservative backlink proposals.",
	}}
	candidate := Issue{
		Category:        "Links",
		Severity:        "low",
		Description:     "89 notes have no incoming or outgoing resolved links. Suggested action: Run naudia links to prepare conservative backlink proposals.",
		SuggestedAction: "Run naudia links.",
	}

	if !duplicateIssue(existing, candidate) {
		t.Fatal("expected AI restatement of deterministic issue to be treated as duplicate")
	}
}

func TestDuplicateIssueAllowsNewSameCategoryFinding(t *testing.T) {
	existing := []Issue{{
		Category:    "Links",
		Severity:    "low",
		Description: "89 notes have no incoming or outgoing resolved links.",
	}}
	candidate := Issue{
		Category:    "Links",
		Severity:    "medium",
		Description: "Several project notes cite the same source note but are not connected to each other.",
	}

	if duplicateIssue(existing, candidate) {
		t.Fatal("expected materially different same-category issue to be kept")
	}
}
