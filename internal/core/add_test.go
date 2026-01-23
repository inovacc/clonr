package core

import "testing"

func TestBytesTrimSpace(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "no whitespace",
			input:    []byte("hello"),
			expected: "hello",
		},
		{
			name:     "leading spaces",
			input:    []byte("  hello"),
			expected: "hello",
		},
		{
			name:     "trailing spaces",
			input:    []byte("hello  "),
			expected: "hello",
		},
		{
			name:     "both sides spaces",
			input:    []byte("  hello  "),
			expected: "hello",
		},
		{
			name:     "newlines",
			input:    []byte("\nhello\n"),
			expected: "hello",
		},
		{
			name:     "carriage returns",
			input:    []byte("\r\nhello\r\n"),
			expected: "hello",
		},
		{
			name:     "tabs",
			input:    []byte("\thello\t"),
			expected: "hello",
		},
		{
			name:     "mixed whitespace",
			input:    []byte(" \t\n\rhello \t\n\r"),
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    []byte("   \t\n\r  "),
			expected: "",
		},
		{
			name:     "internal whitespace preserved",
			input:    []byte("  hello world  "),
			expected: "hello world",
		},
		{
			name:     "url with newline",
			input:    []byte("https://github.com/user/repo\n"),
			expected: "https://github.com/user/repo",
		},
		{
			name:     "git url output",
			input:    []byte("  git@github.com:user/repo.git\r\n"),
			expected: "git@github.com:user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesTrimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("bytesTrimSpace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddOptions_Fields(t *testing.T) {
	opts := AddOptions{
		Yes:  true,
		Name: "test-repo",
	}

	if !opts.Yes {
		t.Error("AddOptions.Yes = false, want true")
	}

	if opts.Name != "test-repo" {
		t.Errorf("AddOptions.Name = %q, want %q", opts.Name, "test-repo")
	}
}

func TestAddOptions_ZeroValue(t *testing.T) {
	var opts AddOptions

	if opts.Yes {
		t.Error("zero AddOptions.Yes = true, want false")
	}

	if opts.Name != "" {
		t.Errorf("zero AddOptions.Name = %q, want empty", opts.Name)
	}
}

func TestAddRepo_EmptyPath(t *testing.T) {
	_, err := AddRepo("", AddOptions{})
	if err == nil {
		t.Error("AddRepo with empty path should return error")
	}

	expected := "path is required"
	if err.Error() != expected {
		t.Errorf("AddRepo(\"\") error = %q, want %q", err.Error(), expected)
	}
}
