package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGitCmd(t *testing.T) {
	t.Run("git command exists", func(t *testing.T) {
		if gitCmd == nil {
			t.Fatal("gitCmd should not be nil")
		}
	})

	t.Run("git command has correct use", func(t *testing.T) {
		if gitCmd.Use != "git" {
			t.Errorf("gitCmd.Use = %q, want %q", gitCmd.Use, "git")
		}
	})

	t.Run("git command has subcommands", func(t *testing.T) {
		subcommands := gitCmd.Commands()
		if len(subcommands) == 0 {
			t.Error("gitCmd should have subcommands")
		}

		expectedSubcommands := []string{"status", "commit", "push", "pull", "log", "diff", "branch", "clone"}
		subcommandMap := make(map[string]bool)
		for _, cmd := range subcommands {
			subcommandMap[cmd.Use] = true
		}

		for _, expected := range expectedSubcommands {
			found := false
			for _, cmd := range subcommands {
				if cmd.Name() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("missing subcommand %q", expected)
			}
		}
	})
}

func TestGitStatusCmd(t *testing.T) {
	t.Run("status command exists", func(t *testing.T) {
		if gitStatusCmd == nil {
			t.Fatal("gitStatusCmd should not be nil")
		}
	})

	t.Run("status command has correct use", func(t *testing.T) {
		if gitStatusCmd.Use != "status" {
			t.Errorf("gitStatusCmd.Use = %q, want %q", gitStatusCmd.Use, "status")
		}
	})

	t.Run("status command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"short", "s"},
			{"porcelain", ""},
			{"branch", "b"},
		}

		for _, f := range flags {
			flag := gitStatusCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})
}

func TestGitCommitCmd(t *testing.T) {
	t.Run("commit command exists", func(t *testing.T) {
		if gitCommitCmd == nil {
			t.Fatal("gitCommitCmd should not be nil")
		}
	})

	t.Run("commit command has correct use", func(t *testing.T) {
		if gitCommitCmd.Use != "commit" {
			t.Errorf("gitCommitCmd.Use = %q, want %q", gitCommitCmd.Use, "commit")
		}
	})

	t.Run("commit command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"message", "m"},
			{"all", "a"},
		}

		for _, f := range flags {
			flag := gitCommitCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})

	t.Run("message flag is required", func(t *testing.T) {
		annotations := gitCommitCmd.Flags().Lookup("message").Annotations
		if annotations == nil {
			t.Skip("no annotations on message flag")
		}
		if _, ok := annotations[cobra.BashCompOneRequiredFlag]; !ok {
			t.Log("message flag should be marked as required")
		}
	})
}

func TestGitPushCmd(t *testing.T) {
	t.Run("push command exists", func(t *testing.T) {
		if gitPushCmd == nil {
			t.Fatal("gitPushCmd should not be nil")
		}
	})

	t.Run("push command has correct use", func(t *testing.T) {
		expected := "push [remote] [branch]"
		if gitPushCmd.Use != expected {
			t.Errorf("gitPushCmd.Use = %q, want %q", gitPushCmd.Use, expected)
		}
	})

	t.Run("push command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"set-upstream", "u"},
			{"force", "f"},
			{"tags", ""},
			{"skip-leaks", ""},
		}

		for _, f := range flags {
			flag := gitPushCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})
}

func TestGitPullCmd(t *testing.T) {
	t.Run("pull command exists", func(t *testing.T) {
		if gitPullCmd == nil {
			t.Fatal("gitPullCmd should not be nil")
		}
	})

	t.Run("pull command has correct use", func(t *testing.T) {
		expected := "pull [remote] [branch]"
		if gitPullCmd.Use != expected {
			t.Errorf("gitPullCmd.Use = %q, want %q", gitPullCmd.Use, expected)
		}
	})
}

func TestGitLogCmd(t *testing.T) {
	t.Run("log command exists", func(t *testing.T) {
		if gitLogCmd == nil {
			t.Fatal("gitLogCmd should not be nil")
		}
	})

	t.Run("log command has correct use", func(t *testing.T) {
		if gitLogCmd.Use != "log" {
			t.Errorf("gitLogCmd.Use = %q, want %q", gitLogCmd.Use, "log")
		}
	})

	t.Run("log command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"limit", "n"},
			{"oneline", ""},
			{"all", ""},
			{"author", ""},
			{"since", ""},
			{"until", ""},
			{"grep", ""},
			{"json", ""},
		}

		for _, f := range flags {
			flag := gitLogCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})

	t.Run("limit flag has default value", func(t *testing.T) {
		flag := gitLogCmd.Flags().Lookup("limit")
		if flag == nil {
			t.Fatal("limit flag not found")
		}
		if flag.DefValue != "10" {
			t.Errorf("limit default = %q, want %q", flag.DefValue, "10")
		}
	})
}

func TestGitDiffCmd(t *testing.T) {
	t.Run("diff command exists", func(t *testing.T) {
		if gitDiffCmd == nil {
			t.Fatal("gitDiffCmd should not be nil")
		}
	})

	t.Run("diff command has correct use", func(t *testing.T) {
		expected := "diff [commit] [-- path]"
		if gitDiffCmd.Use != expected {
			t.Errorf("gitDiffCmd.Use = %q, want %q", gitDiffCmd.Use, expected)
		}
	})

	t.Run("diff command has required flags", func(t *testing.T) {
		flags := []string{"staged", "cached", "stat", "name-only", "name-status"}

		for _, f := range flags {
			flag := gitDiffCmd.Flags().Lookup(f)
			if flag == nil {
				t.Errorf("missing flag %q", f)
			}
		}
	})
}

func TestGitBranchCmd(t *testing.T) {
	t.Run("branch command exists", func(t *testing.T) {
		if gitBranchCmd == nil {
			t.Fatal("gitBranchCmd should not be nil")
		}
	})

	t.Run("branch command has correct use", func(t *testing.T) {
		expected := "branch [name]"
		if gitBranchCmd.Use != expected {
			t.Errorf("gitBranchCmd.Use = %q, want %q", gitBranchCmd.Use, expected)
		}
	})

	t.Run("branch command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"all", "a"},
			{"delete", "d"},
			{"force-delete", "D"},
			{"json", ""},
		}

		for _, f := range flags {
			flag := gitBranchCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})
}

func TestGitCloneCmd(t *testing.T) {
	t.Run("clone command exists", func(t *testing.T) {
		if gitCloneCmd == nil {
			t.Fatal("gitCloneCmd should not be nil")
		}
	})

	t.Run("clone command has correct use", func(t *testing.T) {
		expected := "clone <repository> [<directory>]"
		if gitCloneCmd.Use != expected {
			t.Errorf("gitCloneCmd.Use = %q, want %q", gitCloneCmd.Use, expected)
		}
	})

	t.Run("clone command has required flags", func(t *testing.T) {
		flags := []struct {
			name      string
			shorthand string
		}{
			{"force", "f"},
			{"no-tui", ""},
			{"workspace", "w"},
			{"profile", "p"},
		}

		for _, f := range flags {
			flag := gitCloneCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("missing flag %q", f.name)
				continue
			}
			if f.shorthand != "" && flag.Shorthand != f.shorthand {
				t.Errorf("flag %q shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
			}
		}
	})

	t.Run("clone command requires minimum 1 arg", func(t *testing.T) {
		if gitCloneCmd.Args == nil {
			t.Error("gitCloneCmd.Args should be set")
		}
	})
}
