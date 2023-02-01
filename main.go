package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	starBorder = lipgloss.Border{
		Top:         "*",
		Bottom:      "*",
		Left:        "*",
		Right:       "*",
		TopLeft:     "*",
		TopRight:    "*",
		BottomLeft:  "*",
		BottomRight: "*",
	}
	titleStyle              = lipgloss.NewStyle().MarginLeft(2)
	itemStyle               = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle       = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.AdaptiveColor{Light: "#179299 	", Dark: "#94e2d5"})
	activeItemStyle         = lipgloss.NewStyle().PaddingLeft(3).Border(lipgloss.Border(starBorder), false, false, false, true)
	activeSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(lipgloss.AdaptiveColor{Light: "#179299 	", Dark: "#94e2d5"}).Border(lipgloss.Border(starBorder), false, false, false, true)
	paginationStyle         = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle               = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle           = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item struct {
	title, desc string
	selected    bool
}

type context struct {
	name      string
	namespace string
	selected  bool
}

func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}
	if i.selected {
		fn = func(s string) string {
			return activeItemStyle.Render(s)
		}
	}
	if i.selected && index == m.Index() {
		fn = func(s string) string {
			return activeSelectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	error    string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				cmd := exec.Command("kubectl", "config", "use-context", i.title)
				_, err := cmd.Output()
				if err != nil {
					m.error = err.Error()
				}
				m.choice = string(i.title)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("Activating Context: %s", m.choice))
	}
	if m.error != "" {
		return quitTextStyle.Render(m.error)
	}
	if m.quitting {
		return quitTextStyle.Render("Good bye")
	}
	return "\n" + m.list.View()
}

func main() {
	contexts, _ := getContexts()
	var items []list.Item

	for _, context := range contexts {
		items = append(items, item{title: context.name, desc: context.namespace, selected: context.selected})
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Which Context to load?"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func getContexts() ([]context, error) {
	var contexts []context
	cmd := exec.Command("kubectl", "config", "get-contexts", "--no-headers=true")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return contexts, err
	}
	stdout_lines := strings.Split(string(stdout), "\n")

	for _, line := range stdout_lines {
		if line != "" {
			contexts = append(contexts, parse_line(line))
		}
	}
	return contexts, err
}

func parse_line(line string) context {
	var context context
	line = strings.TrimSpace(line)

	space := regexp.MustCompile(`\s+`)
	line = space.ReplaceAllString(line, " ")

	// Check for selection
	if strings.HasPrefix(line, "*") {
		context.selected = true
		line = strings.TrimSpace(strings.Trim(line, "*"))
	} else {
		context.selected = false
	}
	// Check for namespace
	if len(strings.Split(line, " ")) >= 4 {
		context.namespace = strings.Split(line, " ")[3]
	} else {
		context.namespace = ""
	}
	//Add name
	context.name = strings.Split(line, " ")[0]

	return context
}
