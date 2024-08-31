package main

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string
type tickMsg time.Time

const (
	padding  = 2
	maxWidth = 50
)

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func calculatePercent(m model) float64 {
	if len(m.list.Items()) == 0 {
		return 0
	}

	var completed float64
	for _, i := range m.list.Items() {
		if strings.HasPrefix(string(i.(item)), "[x]") {
			completed++
		}
	}

	return completed / float64(len(m.list.Items()))
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type (
	errMsg error
)

type model struct {
	list      list.Model
	textInput textinput.Model
	input     string
	err       error
	percent   float64
	progress  progress.Model
}

func initialModel() model {
	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))

	ti := textinput.New()
	ti.Placeholder = "task"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	items := []list.Item{}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Tasks List"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return model{
		list:      l,
		textInput: ti,
		err:       nil,
		progress:  prog,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		m.percent += 0.25
		if m.percent > 1.0 {
			m.percent = 1.0
			return m, tea.Quit
		}
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			if len(m.list.Items()) == 0 {
				// Handle the case when the list is empty
				return m, nil
			}
			selected := m.list.SelectedItem().(item)
			var updatedItem item
			if strings.HasPrefix(string(selected), "[x]") {
				updatedItem = item(strings.Replace(string(selected), "[x]", "[ ]", 1))
			} else {
				updatedItem = item(strings.Replace(string(selected), "[ ]", "[x]", 1))
			}
			m.list.SetItem(m.list.Index(), item(updatedItem))
			m.percent = calculatePercent(m)
			return m, nil
		case "ctrl+d":
			if len(m.list.Items()) == 0 {
				// Handle the case when the list is empty
				return m, nil
			}
			m.list.RemoveItem(m.list.Index())
			m.percent = calculatePercent(m)
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.input = m.textInput.Value()
			taskFormatted := fmt.Sprintf("[ ] %s", m.input)
			m.list.InsertItem(99999, item(taskFormatted))
			m.textInput.SetValue("")
			m.percent = calculatePercent(m)
			return m, nil
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	pad := strings.Repeat(" ", padding)
	return m.list.View() +
		"\n" +
		pad + m.progress.ViewAs(m.percent) +
		"\n\n" +
		fmt.Sprintf(
			"Enter Task\n\n%s\n\n%s",
			m.textInput.View(),
			"(esc to quit)",
		) + "\n"
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
