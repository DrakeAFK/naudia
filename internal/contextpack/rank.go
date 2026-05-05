package contextpack

import (
	"sort"
	"strings"
)

var methodWeight = map[string]float64{
	"user_supplied": 100,
	"exact_path":    95,
	"exact_title":   90,
	"title_token":   86,
	"path_token":    84,
	"backlink":      80,
	"outlink":       78,
	"heading_token": 74,
	"folder":        72,
	"tag":           68,
	"token_search":  64,
	"recent_daily":  60,
	"text_search":   50,
	"small_vault":   42,
	"semantic":      35,
}

func Rank(items []Item) []Item {
	out := append([]Item(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		wi := methodWeight[out[i].RetrievalMethod] + out[i].Score
		wj := methodWeight[out[j].RetrievalMethod] + out[j].Score
		if wi == wj {
			if out[i].NotePath == out[j].NotePath {
				return len(out[i].Excerpt) < len(out[j].Excerpt)
			}
			return strings.Compare(out[i].NotePath, out[j].NotePath) < 0
		}
		return wi > wj
	})
	return out
}
