package core

import (
	"fmt"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

// EditorInfo represents editor information for display.
type EditorInfo struct {
	Name    string
	Command string
	Icon    string
}

// DefaultEditors is a list of common editors to check for.
var DefaultEditors = []EditorInfo{
	{Name: "VS Code", Command: "code", Icon: "󰨞"},
	{Name: "Cursor", Command: "cursor", Icon: "󰨞"},
	{Name: "Vim", Command: "vim", Icon: ""},
	{Name: "Neovim", Command: "nvim", Icon: ""},
	{Name: "GoLand", Command: "goland", Icon: ""},
	{Name: "IntelliJ IDEA", Command: "idea", Icon: ""},
	{Name: "WebStorm", Command: "webstorm", Icon: ""},
	{Name: "PyCharm", Command: "pycharm", Icon: ""},
	{Name: "Sublime Text", Command: "subl", Icon: ""},
	{Name: "Atom", Command: "atom", Icon: ""},
	{Name: "Nano", Command: "nano", Icon: ""},
	{Name: "Emacs", Command: "emacs", Icon: ""},
	{Name: "Helix", Command: "hx", Icon: ""},
	{Name: "Zed", Command: "zed", Icon: ""},
}

// AddCustomEditor adds a custom editor to the configuration.
func AddCustomEditor(editor model.Editor) error {
	if editor.Name == "" {
		return fmt.Errorf("editor name is required")
	}

	if editor.Command == "" {
		return fmt.Errorf("editor command is required")
	}

	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := client.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Check if editor already exists
	for _, e := range cfg.CustomEditors {
		if e.Name == editor.Name {
			return fmt.Errorf("editor %q already exists", editor.Name)
		}
	}

	// Check if command already exists in custom editors
	for _, e := range cfg.CustomEditors {
		if e.Command == editor.Command {
			return fmt.Errorf("editor with command %q already exists as %q", editor.Command, e.Name)
		}
	}

	cfg.CustomEditors = append(cfg.CustomEditors, editor)

	if err := client.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// RemoveCustomEditor removes a custom editor from the configuration.
func RemoveCustomEditor(name string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := client.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Find and remove the editor
	found := false
	newEditors := make([]model.Editor, 0, len(cfg.CustomEditors))

	for _, e := range cfg.CustomEditors {
		if e.Name == name {
			found = true

			continue
		}

		newEditors = append(newEditors, e)
	}

	if !found {
		return fmt.Errorf("editor %q not found in custom editors", name)
	}

	cfg.CustomEditors = newEditors

	if err := client.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetCustomEditors returns the list of custom editors from configuration.
func GetCustomEditors() ([]model.Editor, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := client.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return cfg.CustomEditors, nil
}

// GetAllEditors returns all editors (default + custom).
func GetAllEditors() ([]EditorInfo, error) {
	customEditors, err := GetCustomEditors()
	if err != nil {
		return nil, err
	}

	// Start with default editors
	allEditors := make([]EditorInfo, len(DefaultEditors))
	copy(allEditors, DefaultEditors)

	// Add custom editors
	for _, e := range customEditors {
		allEditors = append(allEditors, EditorInfo{
			Name:    e.Name,
			Command: e.Command,
			Icon:    e.Icon,
		})
	}

	return allEditors, nil
}

// GetInstalledEditors returns only installed editors (default + custom).
func GetInstalledEditors() ([]EditorInfo, error) {
	allEditors, err := GetAllEditors()
	if err != nil {
		return nil, err
	}

	var installed []EditorInfo

	for _, editor := range allEditors {
		if IsEditorInstalled(editor.Command) {
			installed = append(installed, editor)
		}
	}

	return installed, nil
}
