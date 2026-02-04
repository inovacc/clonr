package cmd

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/inovacc/clonr/internal/security"
)

func TestGitPushScanModel(t *testing.T) {
	t.Run("newGitPushScanModel initializes correctly", func(t *testing.T) {
		m := newGitPushScanModel()

		if !m.scanning {
			t.Error("scanning should be true initially")
		}
		if m.done {
			t.Error("done should be false initially")
		}
		if m.result != nil {
			t.Error("result should be nil initially")
		}
		if m.err != nil {
			t.Error("err should be nil initially")
		}
	})

	t.Run("Init returns batch command", func(t *testing.T) {
		m := newGitPushScanModel()
		cmd := m.Init()

		if cmd == nil {
			t.Error("Init should return a command")
		}
	})

	t.Run("View shows scanning state", func(t *testing.T) {
		m := newGitPushScanModel()
		view := m.View()

		if view == "" {
			t.Error("View should not be empty when scanning")
		}
	})

	t.Run("View shows done state with no leaks", func(t *testing.T) {
		m := gitPushScanModel{
			done:     true,
			scanning: false,
			result:   &security.ScanResult{HasLeaks: false},
		}

		view := m.View()
		if view == "" {
			t.Error("View should not be empty when done")
		}
	})

	t.Run("View shows done state with leaks", func(t *testing.T) {
		m := gitPushScanModel{
			done:     true,
			scanning: false,
			result: &security.ScanResult{
				HasLeaks: true,
				Findings: []security.Finding{{Description: "test"}},
			},
		}

		view := m.View()
		if view == "" {
			t.Error("View should not be empty when leaks found")
		}
	})

	t.Run("View shows error state", func(t *testing.T) {
		m := gitPushScanModel{
			done:     true,
			scanning: false,
			err:      errTest,
		}

		view := m.View()
		if view == "" {
			t.Error("View should not be empty on error")
		}
	})

	t.Run("Update handles gitPushScanDoneMsg", func(t *testing.T) {
		m := newGitPushScanModel()
		result := &security.ScanResult{HasLeaks: false}

		newModel, cmd := m.Update(gitPushScanDoneMsg{result: result})

		updated := newModel.(gitPushScanModel)
		if !updated.done {
			t.Error("done should be true after receiving done message")
		}
		if updated.scanning {
			t.Error("scanning should be false after receiving done message")
		}
		if updated.result != result {
			t.Error("result should be set")
		}
		if cmd == nil {
			t.Error("should return quit command")
		}
	})

	t.Run("Update handles spinner tick", func(t *testing.T) {
		m := newGitPushScanModel()

		newModel, _ := m.Update(spinner.TickMsg{})

		updated := newModel.(gitPushScanModel)
		if !updated.scanning {
			t.Error("scanning should still be true")
		}
	})
}

// errTest is a test error
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
