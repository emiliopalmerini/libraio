package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"libraio/internal/adapters/editor"
	"libraio/internal/adapters/obsidian"
	"libraio/internal/adapters/tui/views"
	"libraio/internal/ports"
)

func formatMovedMessage(count int) string {
	if count == 0 {
		return "No files moved"
	}
	if count == 1 {
		return "1 file moved"
	}
	return fmt.Sprintf("%d files moved", count)
}

// ViewState represents the current view
type ViewState int

const (
	ViewBrowser ViewState = iota
	ViewCreate
	ViewMove
	ViewArchive
	ViewDelete
	ViewSmartCatalog
	ViewHelp
)

// App is the main TUI application model
type App struct {
	repo      ports.VaultRepository
	editor    *editor.Opener
	obsidian  *obsidian.Opener
	assistant ports.AIAssistant

	state        ViewState
	browser      *views.BrowserModel
	create       *views.CreateModel
	move         *views.MoveModel
	archive      *views.ArchiveModel
	delete       *views.DeleteModel
	smartCatalog *views.SmartCatalogModel
	help         *views.HelpModel

	width  int
	height int
}

// NewApp creates a new TUI application
func NewApp(repo ports.VaultRepository, ed *editor.Opener, obs *obsidian.Opener, assistant ports.AIAssistant) *App {
	return &App{
		repo:         repo,
		editor:       ed,
		obsidian:     obs,
		assistant:    assistant,
		state:        ViewBrowser,
		browser:      views.NewBrowserModel(repo),
		create:       views.NewCreateModel(repo, ed != nil),
		move:         views.NewMoveModel(repo),
		archive:      views.NewArchiveModel(repo),
		delete:       views.NewDeleteModel(repo),
		smartCatalog: views.NewSmartCatalogModel(repo, assistant),
		help:         views.NewHelpModel(),
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
		a.move.SetSize(msg.Width, msg.Height)
		a.archive.SetSize(msg.Width, msg.Height)
		a.delete.SetSize(msg.Width, msg.Height)
		a.smartCatalog.SetSize(msg.Width, msg.Height)
		a.help.SetSize(msg.Width, msg.Height)
		return a, nil

	// View switching messages
	case views.SwitchToCreateMsg:
		a.state = ViewCreate
		a.create.SetParent(msg.ParentNode)
		return a, a.create.Init()

	case views.SwitchToMoveMsg:
		a.state = ViewMove
		a.move.SetSource(msg.SourceNode)
		return a, a.move.Init()

	case views.SwitchToArchiveMsg:
		a.state = ViewArchive
		a.archive.SetTarget(msg.TargetNode)
		return a, a.archive.Init()

	case views.SwitchToDeleteMsg:
		a.state = ViewDelete
		a.delete.SetTarget(msg.TargetNode)
		return a, a.delete.Init()

	case views.SwitchToSmartCatalogMsg:
		a.state = ViewSmartCatalog
		a.smartCatalog.SetSource(msg.SourceNode)
		return a, a.smartCatalog.Init()

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
		return a, a.browser.Reload()

	case views.CreateErrMsg:
		a.create.SetMessage(msg.Err.Error(), true)
		return a, nil

	// Move view messages
	case views.MoveSuccessMsg:
		a.state = ViewBrowser
		return a, a.browser.Reload()

	case views.MoveErrMsg:
		a.move.SetMessage(msg.Err.Error(), true)
		return a, nil

	// Archive view messages
	case views.ArchiveSuccessMsg:
		a.state = ViewBrowser
		return a, a.browser.Reload()

	case views.ArchiveErrMsg:
		// Return to browser on error
		a.state = ViewBrowser
		return a, nil

	// Delete view messages
	case views.DeleteSuccessMsg:
		a.state = ViewBrowser
		return a, a.browser.Reload()

	case views.DeleteErrMsg:
		// Return to browser on error (delete view has no SetMessage)
		a.state = ViewBrowser
		return a, nil

	// Smart catalog view messages
	case views.SmartCatalogFileMoved:
		// File was moved, update the moved count and remove from list
		a.smartCatalog.HandleFileMoved()
		// Check if all files have been processed
		if a.smartCatalog.IsEmpty() {
			a.state = ViewBrowser
			a.browser.SetMessage(formatMovedMessage(a.smartCatalog.GetMovedCount()), false)
			return a, a.browser.Reload()
		}
		return a, nil

	case views.SmartCatalogDoneMsg:
		// All done, return to browser
		a.state = ViewBrowser
		a.browser.SetMessage(formatMovedMessage(msg.Moved), false)
		return a, a.browser.Reload()

	case views.SmartCatalogErrMsg:
		// Return to browser on error
		a.state = ViewBrowser
		return a, nil

	case views.OpenEditorMsg:
		// Return to browser, then open editor
		a.state = ViewBrowser
		return a, a.openEditor(msg.Path)

	case views.OpenObsidianMsg:
		// Open file in Obsidian
		a.state = ViewBrowser
		return a, a.openObsidian(msg.Path)

	case editorFinishedMsg:
		// Reload tree after editor closes to show new/modified items
		return a, a.browser.Reload()
	}

	// Delegate to current view
	var cmd tea.Cmd
	switch a.state {
	case ViewBrowser:
		_, cmd = a.browser.Update(msg)
	case ViewCreate:
		_, cmd = a.create.Update(msg)
	case ViewMove:
		_, cmd = a.move.Update(msg)
	case ViewArchive:
		_, cmd = a.archive.Update(msg)
	case ViewDelete:
		_, cmd = a.delete.Update(msg)
	case ViewSmartCatalog:
		_, cmd = a.smartCatalog.Update(msg)
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

func (a *App) openObsidian(path string) tea.Cmd {
	if a.obsidian == nil {
		return nil
	}

	return func() tea.Msg {
		err := a.obsidian.OpenFile(path)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		return nil
	}
}

// View renders the current view
func (a *App) View() string {
	switch a.state {
	case ViewCreate:
		return a.create.View()
	case ViewMove:
		return a.move.View()
	case ViewArchive:
		return a.archive.View()
	case ViewDelete:
		return a.delete.View()
	case ViewSmartCatalog:
		return a.smartCatalog.View()
	case ViewHelp:
		return a.help.View()
	default:
		return a.browser.View()
	}
}
