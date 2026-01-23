package ports

import "os/exec"

// EditorOpener defines the interface for opening files in an external editor
type EditorOpener interface {
	// OpenFile opens the specified file in the user's preferred editor
	// It uses $EDITOR environment variable, falling back to common editors
	OpenFile(path string) error

	// Command returns an exec.Cmd for opening a file in the editor
	// This is useful for integrating with bubbletea's ExecProcess
	Command(path string) (*exec.Cmd, error)
}
