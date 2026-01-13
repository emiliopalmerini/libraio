package ports

// EditorOpener defines the interface for opening files in an external editor
type EditorOpener interface {
	// OpenFile opens the specified file in the user's preferred editor
	// It uses $EDITOR environment variable, falling back to common editors
	OpenFile(path string) error
}
