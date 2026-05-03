package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCardWrapsLongRowValues(t *testing.T) {
	longSummary := strings.Repeat("review signal ", 12)
	out := Card("Vault Review", [][2]string{{"Summary", longSummary}})

	for _, line := range strings.Split(out, "\n") {
		if width := lipgloss.Width(line); width > 120 {
			t.Fatalf("card line width = %d, want <= 120: %q", width, line)
		}
	}
}
