package cmd

import (
	"testing"

	"github.com/inovacc/clonr/internal/model"
)

func TestFormatTokenStorage(t *testing.T) {
	tests := []struct {
		name     string
		input    model.TokenStorage
		expected string
	}{
		{
			name:     "encrypted storage",
			input:    model.TokenStorageEncrypted,
			expected: "encrypted (TPM)",
		},
		{
			name:     "open storage",
			input:    model.TokenStorageOpen,
			expected: "plain text",
		},
		{
			name:     "unknown storage",
			input:    model.TokenStorage("custom"),
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTokenStorage(tt.input)
			if result != tt.expected {
				t.Errorf("formatTokenStorage(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty path",
			input:   "",
			wantErr: true,
		},
		{
			name:    "absolute path",
			input:   "/tmp/test",
			wantErr: false,
		},
		{
			name:    "home path",
			input:   "~/test",
			wantErr: false,
		},
		{
			name:    "relative path",
			input:   "test/path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Errorf("expandPath(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestCenterString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "string shorter than width",
			input:    "test",
			width:    10,
			expected: "   test   ",
		},
		{
			name:     "string equal to width",
			input:    "test",
			width:    4,
			expected: "test",
		},
		{
			name:     "string longer than width",
			input:    "testing",
			width:    4,
			expected: "testing",
		},
		{
			name:     "odd padding",
			input:    "ab",
			width:    5,
			expected: " ab  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := centerString(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("centerString(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "test",
			maxLen:   10,
			expected: "test",
		},
		{
			name:     "string equal to max",
			input:    "test",
			maxLen:   4,
			expected: "test",
		},
		{
			name:     "string longer than max",
			input:    "testing",
			maxLen:   5,
			expected: "te...",
		},
		{
			name:     "max length 3",
			input:    "testing",
			maxLen:   3,
			expected: "tes",
		},
		{
			name:     "max length 2",
			input:    "testing",
			maxLen:   2,
			expected: "te",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestBoxWidth(t *testing.T) {
	if boxWidth != 64 {
		t.Errorf("boxWidth = %d, want 64", boxWidth)
	}
}

func TestPrintBoxFunctions(t *testing.T) {
	// These functions print to stdout, so we just verify they don't panic
	t.Run("printBoxHeader", func(t *testing.T) {
		// Should not panic
		printBoxHeader("Test Title")
	})

	t.Run("printBoxLine", func(t *testing.T) {
		// Should not panic
		printBoxLine("Label", "Value")
	})

	t.Run("printBoxLine with long content", func(t *testing.T) {
		// Should not panic with content longer than box width
		printBoxLine("Very Long Label", "This is a very long value that exceeds the box width")
	})

	t.Run("printBoxFooter", func(t *testing.T) {
		// Should not panic
		printBoxFooter()
	})

	t.Run("printEmptyResult", func(t *testing.T) {
		// Should not panic
		printEmptyResult("profiles", "clonr profile add")
	})

	t.Run("printInfoBox", func(t *testing.T) {
		// Should not panic
		items := map[string]string{
			"Name":   "test",
			"Status": "active",
		}
		order := []string{"Name", "Status"}
		printInfoBox("Test Box", items, order)
	})

	t.Run("printInfoBox with missing key", func(t *testing.T) {
		// Should not panic with missing key in order
		items := map[string]string{
			"Name": "test",
		}
		order := []string{"Name", "Missing"}
		printInfoBox("Test Box", items, order)
	})
}
