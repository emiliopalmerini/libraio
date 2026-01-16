package obsidian

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Opener implements ports.ObsidianOpener
type Opener struct {
	vaultPath string
	vaultName string
}

// NewOpener creates a new Obsidian opener for the given vault path
func NewOpener(vaultPath string) *Opener {
	vaultName := filepath.Base(vaultPath)
	return &Opener{
		vaultPath: vaultPath,
		vaultName: vaultName,
	}
}

// OpenFile opens a file in Obsidian using the obsidian:// URI scheme
func (o *Opener) OpenFile(filePath string) error {
	uri, err := o.BuildURI(filePath)
	if err != nil {
		return err
	}
	return o.openURI(uri)
}

// BuildURI constructs the obsidian:// URI for a given file path
func (o *Opener) BuildURI(filePath string) (string, error) {
	relPath, err := filepath.Rel(o.vaultPath, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("file is outside the vault: %s", filePath)
	}

	// Obsidian expects forward slashes in paths
	relPath = filepath.ToSlash(relPath)

	uri := fmt.Sprintf("obsidian://open?vault=%s&file=%s",
		url.QueryEscape(o.vaultName),
		url.QueryEscape(relPath),
	)

	return uri, nil
}

func (o *Opener) openURI(uri string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", uri)
	case "linux":
		cmd = exec.Command("xdg-open", uri)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", uri)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Run()
}
