package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			Padding(1, 2).
			Margin(1)
	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("0"))
	// change to whatever color you want
	regularItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212"))
	// default style for items
)

type keymap struct {
	next, prev, runSearch, quit key.Binding
}

type model struct {
	width         int
	height        int
	keymap        keymap
	help          help.Model
	input         textinput.Model
	outputs       [2]viewport.Model
	markdownFiles []MarkdownFile

	searchResults []MarkdownFile
	selectedIndex int
}

type MarkdownFile struct {
	Path    string
	Content string
}

func readMarkdownFiles(dir string) ([]MarkdownFile, error) {
	var markdownFiles []MarkdownFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			mdFile := MarkdownFile{Path: path, Content: string(content)}
			markdownFiles = append(markdownFiles, mdFile)
		}
		return nil
	})
	return markdownFiles, err
}

func searchMarkdownFiles(files []MarkdownFile, term string) []MarkdownFile {
	var results []MarkdownFile
	for _, mdFile := range files {
		if strings.Contains(mdFile.Path, term) || strings.Contains(mdFile.Content, term) {
			results = append(results, mdFile)
		}
	}
	return results
}

func newTextarea() textinput.Model {
	t := textinput.New()
	t.Prompt = ""
	t.Placeholder = "Type your search term and press Enter..."
	t.Cursor.Style = cursorStyle
	t.KeyMap.DeleteWordBackward.SetEnabled(false)
	t.Blur()
	return t
}

func newModel(files []MarkdownFile) model {
	vp1 := viewport.New(100, 30)
	vp2 := viewport.New(100, 30)

	m := model{
		input:         newTextarea(),
		outputs:       [2]viewport.Model{vp1, vp2},
		markdownFiles: files,
		help:          help.New(),
		keymap: keymap{
			next: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next"),
			),
			prev: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "prev"),
			),
			runSearch: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "run search"),
			),
			quit: key.NewBinding(
				key.WithKeys("esc", "ctrl+c"),
				key.WithHelp("esc", "quit"),
			),
		},
	}
	m.input.Focus()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.input.Blur()
			return m, tea.Quit
		case key.Matches(msg, m.keymap.next):
			if len(m.searchResults) > 0 {
				m.selectedIndex = (m.selectedIndex + 1) % len(m.searchResults)
				m.outputs[1].SetContent(m.searchResults[m.selectedIndex].Content)
			}
		case key.Matches(msg, m.keymap.prev):
			if len(m.searchResults) > 0 {
				m.selectedIndex = (m.selectedIndex - 1 + len(m.searchResults)) % len(m.searchResults)
				m.outputs[1].SetContent(m.searchResults[m.selectedIndex].Content)
			}
		case key.Matches(msg, m.keymap.runSearch):
			if term := strings.TrimSpace(m.input.Value()); term != "" {
				// Perform search when Enter is pressed
				m.searchResults = searchMarkdownFiles(m.markdownFiles, term)
				m.selectedIndex = 0 // reset to first result
				if len(m.searchResults) > 0 {
					m.outputs[0].SetContent(formatSearchResults(m.searchResults))
					m.outputs[1].SetContent(m.searchResults[m.selectedIndex].Content)
				} else {
					m.outputs[0].SetContent("No results found.")
					m.outputs[1].SetContent("")
				}
			}
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)

	return m, tea.Batch(inputCmd)
}

// Helper to format search results for displaying in viewport 1
func formatSearchResults(results []MarkdownFile) string {
	var resultsView string
	for i, file := range results {
		if i == 0 { // Update to highlight the first item as selected by default
			resultsView += selectedItemStyle.Render(file.Path) + "\n"
		} else {
			resultsView += regularItemStyle.Render(file.Path) + "\n"
		}
	}
	return borderStyle.Render(resultsView) // Render the whole view with a border
}

func (m model) View() string {
	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.next,
		m.keymap.prev,
		m.keymap.runSearch,
		m.keymap.quit,
	})
	// Prepare the display for viewport 0 (the search results)
	m.outputs[0].SetContent(formatSearchResults(m.searchResults))

	// Viewport 1 still holds the content of the selected markdown file
	typedInput := m.input.View()

	views := []string{
		m.outputs[0].View(),
		m.outputs[1].View(),
	}

	return fmt.Sprintf(
		"%s\n\n%s",
		typedInput,
		lipgloss.JoinHorizontal(lipgloss.Top, views...)+"\n\n"+help,
	)
}

func main() {
	// Adjust the directory to read markdown files from here
	dir := "/home/stephen/Documents/JCA Personal/"
	files, err := readMarkdownFiles(dir)
	if err != nil {
		fmt.Println("Error reading files:", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(newModel(files), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error while running program:", err)
		os.Exit(1)
	}
}
