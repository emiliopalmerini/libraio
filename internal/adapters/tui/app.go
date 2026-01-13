package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/editor"
	"libraio/internal/adapters/tui/views"
	"libraio/internal/ports"
)

// ViewState represents the current view
type ViewState int

const (
	ViewBrowser ViewState = iota
	ViewCreate
	ViewHelp
)

// App is the main TUI application model
type App struct {
	repo   ports.VaultRepository
	editor *editor.Opener

	state   ViewState
	browser *views.BrowserModel
	create  *views.CreateModel
	help    *views.HelpModel

	width  int
	height int
}

// NewApp creates a new TUI application
func NewApp(repo ports.VaultRepository, ed *editor.Opener) *App {
	return &App{
		repo:    repo,
		editor:  ed,
		state:   ViewBrowser,
		browser: views.NewBrowserModel(repo),
		create:  views.NewCreateModel(repo, ed != nil),
		help:    views.NewHelpModel(),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return a.browser.Init()
}

// Update handles messages for the application
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.browser.SetSize(msg.Width, msg.Height)
		a.create.SetSize(msg.Width, msg.Height)
		a.help.SetSize(msg.Width, msg.Height)
		return a, nil

	// View switching messages
	case views.SwitchToCreateMsg:
		a.state = ViewCreate
		a.create.SetParent(msg.ParentNode)
		return a, a.create.Init()

	case views.SwitchToMoveMsg:
		// For now, just show message - move could be another view
		// TODO: Implement move view
		return a, nil

	case views.SwitchToSearchMsg:
		// Search is now inline in browser, no need to switch
		return a, nil

	case views.SwitchToHelpMsg:
		a.state = ViewHelp
		return a, nil

	case views.SwitchToBrowserMsg:
		a.state = ViewBrowser
		return a, a.browser.Reload()

	// Create view messages
	case views.CreateSuccessMsg:
		a.state = ViewBrowser
		return a, nil

	case views.CreateErrMsg:
		a.create.SetMessage(msg.Err.Error(), true)
		return a, nil

	case views.OpenEditorMsg:
		// Return to browser, then open editor
		a.state = ViewBrowser
		return a, a.openEditor(msg.Path)
	}

	// Delegate to current view
	var cmd tea.Cmd
	switch a.state {
	case ViewBrowser:
		_, cmd = a.browser.Update(msg)
	case ViewCreate:
		_, cmd = a.create.Update(msg)
	case ViewHelp:
		_, cmd = a.help.Update(msg)
	}

	return a, cmd
}

type editorFinishedMsg struct{ err error }

func (a *App) openEditor(path string) tea.Cmd {
	if a.editor == nil {
		return nil
	}

	cmd, err := a.editor.Command(path)
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: err}
		}
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// View renders the current view
func (a *App) View() string {
	switch a.state {
	case ViewCreate:
		return a.create.View()
	case ViewHelp:
		return a.help.View()
	default:
		return a.browser.View()
	}
}
