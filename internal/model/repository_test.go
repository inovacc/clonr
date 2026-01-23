package model

import (
	"testing"
	"time"
)

func TestRepository_Fields(t *testing.T) {
	now := time.Now()

	repo := Repository{
		ID:          1,
		UID:         "abc-123-def",
		URL:         "https://github.com/user/repo",
		Path:        "/home/user/repos/repo",
		Favorite:    true,
		ClonedAt:    now,
		UpdatedAt:   now,
		LastChecked: now,
	}

	if repo.ID != 1 {
		t.Errorf("ID = %d, want %d", repo.ID, 1)
	}

	if repo.UID != "abc-123-def" {
		t.Errorf("UID = %q, want %q", repo.UID, "abc-123-def")
	}

	if repo.URL != "https://github.com/user/repo" {
		t.Errorf("URL = %q, want %q", repo.URL, "https://github.com/user/repo")
	}

	if repo.Path != "/home/user/repos/repo" {
		t.Errorf("Path = %q, want %q", repo.Path, "/home/user/repos/repo")
	}

	if !repo.Favorite {
		t.Error("Favorite = false, want true")
	}

	if !repo.ClonedAt.Equal(now) {
		t.Errorf("ClonedAt = %v, want %v", repo.ClonedAt, now)
	}

	if !repo.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", repo.UpdatedAt, now)
	}

	if !repo.LastChecked.Equal(now) {
		t.Errorf("LastChecked = %v, want %v", repo.LastChecked, now)
	}
}

func TestRepository_ZeroValues(t *testing.T) {
	var repo Repository

	if repo.ID != 0 {
		t.Errorf("zero Repository.ID = %d, want 0", repo.ID)
	}

	if repo.UID != "" {
		t.Errorf("zero Repository.UID = %q, want empty", repo.UID)
	}

	if repo.URL != "" {
		t.Errorf("zero Repository.URL = %q, want empty", repo.URL)
	}

	if repo.Path != "" {
		t.Errorf("zero Repository.Path = %q, want empty", repo.Path)
	}

	if repo.Favorite {
		t.Error("zero Repository.Favorite = true, want false")
	}

	if !repo.ClonedAt.IsZero() {
		t.Errorf("zero Repository.ClonedAt = %v, want zero", repo.ClonedAt)
	}

	if !repo.UpdatedAt.IsZero() {
		t.Errorf("zero Repository.UpdatedAt = %v, want zero", repo.UpdatedAt)
	}

	if !repo.LastChecked.IsZero() {
		t.Errorf("zero Repository.LastChecked = %v, want zero", repo.LastChecked)
	}
}

func TestRepository_Timestamps(t *testing.T) {
	cloned := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	checked := time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)

	repo := Repository{
		ClonedAt:    cloned,
		UpdatedAt:   updated,
		LastChecked: checked,
	}

	// Verify timestamps are independent
	if repo.ClonedAt.After(repo.UpdatedAt) {
		t.Error("ClonedAt should be before UpdatedAt")
	}

	if repo.UpdatedAt.After(repo.LastChecked) {
		t.Error("UpdatedAt should be before LastChecked")
	}
}
