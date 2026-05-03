package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type DiffModel struct {
	viewport viewport.Model
	content  string
}

func NewDiffModel(content string, width, height int) DiffModel {
	vp := viewport.New(width, height)
	vp.SetContent(colorDiff(content))
	return DiffModel{viewport: vp, content: content}
}

func (m DiffModel) Init() tea.Cmd { return nil }

func (m DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 1
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffModel) View() string {
	return m.viewport.View() + "\n" + DefaultTheme().MutedStyle().Render("up/down: scroll  q: quit")
}

func colorDiff(diff string) string {
	theme := DefaultTheme()
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			lines[i] = theme.SuccessStyle().Render(line)
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			lines[i] = theme.DangerStyle().Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
