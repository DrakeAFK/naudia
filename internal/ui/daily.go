package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type DailyModel struct {
	viewport viewport.Model
	title    string
}

func NewDailyModel(title, content string, width, height int) DailyModel {
	vp := viewport.New(width, height)
	vp.SetContent(Markdown(content))
	return DailyModel{viewport: vp, title: title}
}

func (m DailyModel) Init() tea.Cmd { return nil }

func (m DailyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 2
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DailyModel) View() string {
	return DefaultTheme().Title().Render(m.title) + "\n\n" + m.viewport.View() + "\n" + DefaultTheme().MutedStyle().Render("up/down: scroll  q: quit")
}
