package ui

import (
	"fmt"
	"strings"

	"github.com/DrakeAFK/naudia/internal/engines"
	"github.com/DrakeAFK/naudia/internal/proposals"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type issueItem struct {
	issue engines.Issue
}

func (i issueItem) FilterValue() string { return i.issue.Description }
func (i issueItem) Title() string       { return i.issue.Category + " / " + i.issue.Severity }
func (i issueItem) Description() string { return i.issue.Description }

type ReviewModel struct {
	report    engines.Report
	proposals []*proposals.Proposal
	list      list.Model
}

func NewReviewModel(report engines.Report, props []*proposals.Proposal) ReviewModel {
	items := make([]list.Item, 0, len(report.Issues))
	for _, issue := range report.Issues {
		items = append(items, issueItem{issue: issue})
	}
	l := list.New(items, list.NewDefaultDelegate(), 90, 20)
	l.Title = "Vault Review"
	l.SetFilteringEnabled(false)
	return ReviewModel{report: report, proposals: props, list: l}
}

func (m ReviewModel) Init() tea.Cmd { return nil }

func (m ReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 8)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ReviewModel) View() string {
	var b strings.Builder
	b.WriteString(Card("Vault Review", [][2]string{
		{"Summary", m.report.Summary},
		{"Proposals", fmt.Sprintf("%d prepared", len(m.proposals))},
		{"Report", emptyDash(m.report.ReportPath)},
	}))
	b.WriteString("\n\n")
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(KeyHelp())
	return b.String()
}
