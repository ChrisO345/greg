package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	allItems []string
	filtered []string
	cursor   int
	input    string
	width    int
	height   int
	config   *Config

	mode       string
	prompt     string
	out        string
	mainHeader string
	helpText   string

	windowStart int // start index of visible window
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Set cursor to -1 to indicate no selection
			m.cursor = -1
			return m, tea.Quit

		case "enter":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.windowStart {
					m.windowStart--
				}
			}

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.windowStart+m.config.MaxItems {
					m.windowStart++
				}
			}

		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
				m.filterItems()
				m.windowStart = 0
			}

		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
				m.filterItems()
				m.windowStart = 0
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *model) filterItems() {
	if m.input == "" {
		m.filtered = m.allItems
		m.cursor = 0
		m.windowStart = 0
		return
	}
	var f []string
	for _, item := range m.allItems {
		if strings.Contains(strings.ToLower(item), strings.ToLower(m.input)) {
			f = append(f, item)
		}
	}
	m.filtered = f
	if m.cursor >= len(f) {
		m.cursor = 0
	}
	if m.windowStart >= len(f) {
		m.windowStart = 0
	}
}

func (m model) View() string {
	cfg := m.config

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(cfg.Colors.Title))
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Colors.Prompt))
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Colors.Item))
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color(cfg.Colors.Selected)).
		Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Colors.Help))

	header := ""
	if m.mainHeader != "" {
		header = titleStyle.Render(m.mainHeader) + helpStyle.Render(m.helpText) + "\n"
	}
	prompt := fmt.Sprintf("%s %s\n\n", promptStyle.Render(m.prompt), m.input)

	// Determine visible slice based on windowStart
	start := m.windowStart
	end := min(start+cfg.MaxItems, len(m.filtered))
	visibleItems := m.filtered[start:end]

	list := ""
	for i, item := range visibleItems {
		globalIndex := start + i
		if globalIndex == m.cursor {
			list += selectedStyle.Render(" > "+item) + "\n"
		} else {
			list += itemStyle.Render("   "+item) + "\n"
		}
	}

	if len(m.filtered) == 0 {
		list += helpStyle.Render("   no matches found")
	} else if len(m.filtered) > cfg.MaxItems {
		list += helpStyle.Render(fmt.Sprintf("   ...and %d more", len(m.filtered)-cfg.MaxItems))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, prompt, list)
	return lipgloss.NewStyle().Margin(1, 2).Render(content)
}

func RunTUIWithItems(cfg *Config, mode model, items []string, apps []AppEntry) (string, error) {
	p := tea.NewProgram(mode, tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	mod := m.(model)

	if len(mod.filtered) == 0 {
		return "", nil
	}

	if mod.cursor == -1 {
		// No selection made
		return "", nil
	}

	selected := mod.filtered[mod.cursor]

	switch mod.mode {
	case "dmenu":
		if mod.out != "" {
			if err := os.MkdirAll(filepath.Dir(mod.out), 0755); err != nil {
				return "", fmt.Errorf("failed to create output directory: %w", err)
			}
			if err := os.WriteFile(mod.out, []byte(selected+"\n"), 0644); err != nil {
				return "", fmt.Errorf("failed to write selection to file: %w", err)
			}
		} else {
			fmt.Println(selected)
		}
	case "apps":
		// Find the corresponding .desktop file
		for _, app := range apps {
			if app.Name == selected {
				return "", launchDesktopFile(app.Path)
			}
		}
		fmt.Fprintf(os.Stderr, "Error: could not find .desktop file for %s\n", selected)
	case "menu":
		// No action needed; menu handling is done elsewhere
		return selected, nil
	}
	return "", nil
}

// initialModelWithItems is like initialModel but accepts a preloaded list
func initialModelWithItems(cfg *Config, args *CLIArgs, items []string) model {
	prompt := args.Prompt.Value
	if prompt == "" {
		prompt = "search>"
	}

	mainHeader := args.Header.Value
	helpText := ""
	if mainHeader == "" {
		mainHeader = "greg"
		helpText = " - type to filter, ↑↓ to move, enter to select"
	}

	return model{
		allItems: items,
		filtered: items,
		config:   cfg,

		mode:   args.Mode.Value,
		prompt: prompt,
		out:    args.Out.Value,

		mainHeader: mainHeader,
		helpText:   helpText,

		windowStart: 0,
	}
}
