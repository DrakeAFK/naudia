package contextpack

import (
	"fmt"
	"strings"
)

func Render(pack Pack) string {
	var b strings.Builder
	for i, item := range pack.Items {
		fmt.Fprintf(&b, "Source %d\n", i+1)
		fmt.Fprintf(&b, "Path: %s\n", item.NotePath)
		if item.ObsidianURI != "" {
			fmt.Fprintf(&b, "Obsidian: %s\n", item.ObsidianURI)
		}
		if item.Heading != "" {
			fmt.Fprintf(&b, "Heading: %s\n", item.Heading)
		}
		fmt.Fprintf(&b, "Method: %s\n", item.RetrievalMethod)
		fmt.Fprintf(&b, "Score: %.2f\n", item.Score)
		fmt.Fprintf(&b, "Reason: %s\n", item.Reason)
		fmt.Fprintf(&b, "Excerpt:\n%s\n\n", item.Excerpt)
	}
	if pack.BudgetExceeded {
		fmt.Fprintf(&b, "Context budget reached. Selected %d candidates and dropped %d weaker candidates.\n", len(pack.Items), pack.Dropped)
	}
	return strings.TrimSpace(b.String())
}

func Table(pack Pack) string {
	var b strings.Builder
	for _, item := range pack.Items {
		fmt.Fprintf(&b, "%-36s %-14s %.2f %6d %s\n", item.NotePath, item.RetrievalMethod, item.Score, item.CharCount, item.Heading)
	}
	if pack.Dropped > 0 {
		fmt.Fprintf(&b, "\nDropped %d lower-priority context items.\n", pack.Dropped)
	}
	return strings.TrimRight(b.String(), "\n")
}
