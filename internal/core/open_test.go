package core

import (
	"testing"
)

func TestIsEditorInstalled(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		expected bool
	}{
		{
			name:     "bash should be installed",
			editor:   "bash",
			expected: true,
		},
		{
			name:     "nonexistent editor",
			editor:   "nonexistent-editor-that-does-not-exist-12345",
			expected: false,
		},
		{
			name:     "empty string",
			editor:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEditorInstalled(tt.editor)
			if result != tt.expected {
				t.Errorf("IsEditorInstalled(%q) = %v, want %v", tt.editor, result, tt.expected)
			}
		})
	}
}

func TestOpenInFileManager_InvalidPath(t *testing.T) {
	// We can't easily test success cases without actually opening a file manager,
	// but we can verify the function doesn't panic with various inputs.
	// The actual command execution may fail on CI environments without display.

	// Test that it handles paths without crashing
	// Note: On headless systems, this may fail to start the process,
	// but it should not panic.
	_ = OpenInFileManager("/nonexistent/path/that/does/not/exist")
}

func TestOpenInEditor_InvalidEditor(t *testing.T) {
	err := OpenInEditor("nonexistent-editor-12345", "/tmp")
	if err == nil {
		t.Error("expected error for nonexistent editor, got nil")
	}
}
