package core

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenInFileManager opens the given path in the system's default file manager.
func OpenInFileManager(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open file manager: %w", err)
	}

	return nil
}

// OpenInEditor opens the given path in the specified editor.
func OpenInEditor(editor, path string) error {
	cmd := exec.Command(editor, path)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open editor %s: %w", editor, err)
	}

	return nil
}

// IsEditorInstalled checks if the given editor command is available in PATH.
func IsEditorInstalled(editor string) bool {
	_, err := exec.LookPath(editor)

	return err == nil
}
