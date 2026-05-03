package contextpack

import (
	"strings"

	"github.com/drakeafk/naudia/internal/util"
)

func Build(items []Item, budget Budget) Pack {
	if budget.MaxNotes <= 0 {
		budget.MaxNotes = 8
	}
	if budget.MaxChunks <= 0 {
		budget.MaxChunks = 16
	}
	if budget.MaxCharsTotal <= 0 {
		budget.MaxCharsTotal = 24000
	}
	if budget.MaxCharsPerNote <= 0 {
		budget.MaxCharsPerNote = 6000
	}
	ranked := Deduplicate(Rank(items))
	noteChars := map[string]int{}
	notes := map[string]bool{}
	var selected []Item
	var chars int
	var semanticCount int
	for _, item := range ranked {
		item.Excerpt = trimExcerpt(item.Excerpt, budget.MaxCharsPerNote)
		item.CharCount = len(item.Excerpt)
		if item.RetrievalMethod == "semantic" {
			if item.Score < budget.MinSimilarityThreshold {
				continue
			}
			if budget.MaxSemanticMatches > 0 && semanticCount >= budget.MaxSemanticMatches {
				continue
			}
		}
		if !notes[item.NotePath] && len(notes) >= budget.MaxNotes {
			continue
		}
		if len(selected) >= budget.MaxChunks {
			continue
		}
		if chars+item.CharCount > budget.MaxCharsTotal {
			continue
		}
		if noteChars[item.NotePath]+item.CharCount > budget.MaxCharsPerNote {
			continue
		}
		selected = append(selected, item)
		notes[item.NotePath] = true
		noteChars[item.NotePath] += item.CharCount
		chars += item.CharCount
		if item.RetrievalMethod == "semantic" {
			semanticCount++
		}
	}
	return Pack{
		Items:          selected,
		Dropped:        len(ranked) - len(selected),
		SelectedChars:  chars,
		BudgetExceeded: len(selected) < len(ranked),
	}
}

func Deduplicate(items []Item) []Item {
	seen := map[string]bool{}
	var out []Item
	for _, item := range items {
		key := item.NotePath + "|" + item.Heading + "|" + util.SHA256String(strings.ToLower(strings.TrimSpace(item.Excerpt)))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func trimExcerpt(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	cut := max
	if idx := strings.LastIndex(s[:max], "\n"); idx > max/2 {
		cut = idx
	}
	return strings.TrimSpace(s[:cut])
}
