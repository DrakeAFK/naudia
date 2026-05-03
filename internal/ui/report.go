package ui

import (
	"fmt"
	"strings"

	"github.com/drakeafk/naudia/internal/engines"
)

func ReportView(report engines.Report) string {
	rows := [][2]string{
		{"Summary", report.Summary},
	}
	for _, detail := range report.Details {
		rows = append(rows, [2]string{detail.Label, detail.Value})
	}
	if report.ReportPath != "" {
		rows = append(rows, [2]string{"Full report", report.ReportPath})
	}
	out := Card(report.Title, rows)
	if len(report.Issues) > 0 {
		var b strings.Builder
		b.WriteString(out)
		b.WriteString("\n\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(&b, "%s\n  - %s\n", issue.Category, issue.Description)
			if issue.SuggestedAction != "" {
				fmt.Fprintf(&b, "  - %s\n", issue.SuggestedAction)
			}
		}
		return strings.TrimRight(b.String(), "\n")
	}
	if len(report.Lines) > 0 {
		return out + "\n\n" + strings.Join(report.Lines, "\n")
	}
	return out
}
