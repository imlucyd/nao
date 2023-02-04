package config

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luisnquin/nao/v3/internal"
	"github.com/luisnquin/nao/v3/internal/ui"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// Config panel views.
const (
	FileConflict = "File conflict resolution"
	Encryption   = "Encryption"
	Language     = "Language"
	Editor       = "Editor"
	Themes       = "Themes"
	Exit         = "Exit"
)

// File conflict resolution options.
const (
	UseInReadOnlyMode = "Use in read-only mode"
	ExitProgram       = "Exit program"
)

// Basic boolean options.
const (
	Enable  = "Enable"
	Disable = "Disable"
)

type configPanel struct {
	*Core

	list        list.Model
	currentView string
	cursor      int
}

type (
	editorItem struct {
		name   string
		usable bool
	}

	genericItem struct {
		name, desc string
	}
)

func (e editorItem) Title() string { return e.name }
func (e editorItem) Description() string {
	if !e.usable {
		return "not available in $PATH"
	}

	return "available in $PATH"
}
func (e editorItem) FilterValue() string { return e.name }

func (g genericItem) Title() string       { return g.name }
func (g genericItem) Description() string { return g.desc }
func (g genericItem) FilterValue() string { return g.name }

// Creates a new interactive configuration panel.
func InitPanel(core *Core) error {
	p := tea.NewProgram(initConfigPanel(core), tea.WithAltScreen(), tea.WithANSICompressor())

	_, err := p.Run()

	return err
}

func initConfigPanel(core *Core) configPanel {
	delegate := list.NewDefaultDelegate()

	p := configPanel{
		Core: core,
		list: list.New(getDefaultPanelItems(), delegate, 0, 0),
	}

	p.list.Title = "--- Configuration panel ---"

	return p
}

func (c configPanel) Init() tea.Cmd { return nil }

func (c configPanel) View() string { return docStyle.Render(c.list.View()) }

func (c configPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	try := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, 'q', 'Q':
			return c, tea.Quit

		case tea.KeyEsc, tea.KeyLeft:
			cmd := c.list.SetItems(getDefaultPanelItems())
			c.list.ResetSelected()
			c.currentView = ""
			c.cursor = 0

			return c, cmd

		case tea.KeyEnter, tea.KeyRight:
			selectedItem := c.list.VisibleItems()[c.cursor].FilterValue()

			switch c.currentView {
			case Editor:
				if !strings.Contains(selectedItem, "(current)") {
					c.Editor.Name = selectedItem
					try(c.Save())
				}

				return c, tea.Quit

			case Encryption:
				c.Encrypt = selectedItem == Enable
				try(c.Save())

			case Themes:
				if !strings.Contains(selectedItem, "(current)") {
					// c.UpdateTheme(theme)
					c.Theme = selectedItem
					try(c.Save())
				}

				return c, tea.Quit

			case FileConflict:
				c.Core.ReadOnlyOnConflict = selectedItem == UseInReadOnlyMode
				try(c.Save())

				return c, tea.Quit

			case Language:
				return c, tea.Quit

			default:
				switch selectedItem {
				case Editor:
					c.currentView = Editor

					return c, c.list.SetItems(getEditorItems(c.Core))

				case Themes:
					c.currentView = Themes

					return c, c.list.SetItems(getThemeItems(c.Core))

				case FileConflict:
					c.currentView = FileConflict

					return c, c.list.SetItems(getFileConflictResolutionOpts())

				case Language:
					c.currentView = Language

					return c, c.list.SetItems(getLanguageItems())

				case Encryption:
					c.currentView = Encryption

					return c, c.list.SetItems(getEncryptionOptions())

				case Exit:
					return c, tea.Quit

				default:
					panic("unknown panel option")
				}
			}

		case tea.KeyUp, tea.KeyType('k'):
			if c.cursor > 0 {
				c.cursor--
			}

		case tea.KeyDown, tea.KeyType('d'):
			if c.cursor < len(c.list.VisibleItems())-1 {
				c.cursor++
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		c.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)

	return c, cmd
}

func getDefaultPanelItems() []list.Item {
	return []list.Item{
		genericItem{name: FileConflict, desc: "Prevent multiple instances of the same file from opening the file in read-only mode or exit with an error"},
		genericItem{name: Editor, desc: "Select the terminal editor of your preference"},
		genericItem{name: Language, desc: "Set your preferred language for program output"},
		genericItem{name: Encryption, desc: "Protect confidential data files with encryption by using a keyring tool 🔑"},
		genericItem{name: Themes, desc: "Explore dream options 🌌"},
		genericItem{name: Exit, desc: "Secret ending"},
	}
}

func getFileConflictResolutionOpts() []list.Item {
	return []list.Item{
		genericItem{name: UseInReadOnlyMode, desc: "Open the file in read-only mode without editing anything"},
		genericItem{name: ExitProgram, desc: "Only exit with an error when this event is detected"},
	}
}

func getEncryptionOptions() []list.Item {
	return []list.Item{
		genericItem{name: Enable, desc: "Enable boolean to encrypt all your data"},
		genericItem{name: Disable, desc: "Disable and revert encryption in case you already use encryption"},
	}
}

func getEditorItems(c *Core) []list.Item {
	editors := []string{internal.Nano, internal.Neovim, internal.Vim}
	listItems := make([]list.Item, len(editors))

	for i, name := range editors {
		_, err := exec.LookPath(name)
		if name == c.Editor.Name {
			name += ui.GetPrinter(c.Colors.One).Sprint(" (current)")
		}

		listItems[i] = editorItem{
			name:   name,
			usable: err == nil,
		}
	}

	return listItems
}

func getThemeItems(c *Core) []list.Item {
	themes := ui.GetThemes()
	listItems := make([]list.Item, len(themes))

	for i, theme := range themes {
		name := theme.Name

		if name == c.Theme {
			name += ui.GetPrinter(c.Colors.One).Sprint(" (current)")
		}

		listItems[i] = genericItem{
			name: name,
			desc: theme.Pretty(),
		}
	}

	return listItems
}

func getLanguageItems() []list.Item {
	return []list.Item{
		genericItem{
			name: "English - en",
			desc: "whoami",
		},
		genericItem{
			name: "Spanish - es",
			desc: "quiénsoy",
		},
		genericItem{
			name: "French - fr",
			desc: "quisuisje",
		},
	}
}
