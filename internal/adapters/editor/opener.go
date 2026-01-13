package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// Opener implements ports.EditorOpener
type Opener struct{}

// NewOpener creates a new editor opener
func NewOpener() *Opener {
	return &Opener{}
}

// OpenFile opens a file in the user's preferred editor
func (o *Opener) OpenFile(path string) error {
	cmd, err := o.Command(path)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// Command returns an exec.Cmd for opening a file in the editor
// This is useful for integrating with bubbletea's ExecProcess
func (o *Opener) Command(path string) (*exec.Cmd, error) {
	editor := o.findEditor()
	if editor == "" {
		return nil, fmt.Errorf("no editor found: set $EDITOR environment variable")
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, nil
}

// findEditor returns the editor to use
func (o *Opener) findEditor() string {
	// Check $EDITOR first
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Check $VISUAL
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Try common editors
	editors := []string{"nvim", "vim", "vi", "nano", "code"}
	for _, editor := range editors {
		if path, err := exec.LookPath(editor); err == nil {
			return path
		}
	}

	return ""
}
