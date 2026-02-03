package grpc

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/store"
)

// RotationScheduler periodically checks and auto-rotates expired encryption keys.
type RotationScheduler struct {
	store         store.Store
	checkInterval time.Duration
	maxAge        time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.Mutex
	running       bool
}

// NewRotationScheduler creates a new rotation scheduler.
// checkInterval is how often to check for expired keys.
// maxAge is the maximum age before a key is rotated (0 to disable).
func NewRotationScheduler(db store.Store, checkInterval, maxAge time.Duration) *RotationScheduler {
	return &RotationScheduler{
		store:         db,
		checkInterval: checkInterval,
		maxAge:        maxAge,
	}
}

// Start begins the rotation scheduler background task.
func (rs *RotationScheduler) Start() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		return // Already running
	}

	rs.ctx, rs.cancel = context.WithCancel(context.Background())
	rs.running = true

	rs.wg.Add(1)
	go rs.run()

	slog.Info("key rotation scheduler started",
		"check_interval", rs.checkInterval,
		"max_age", rs.maxAge)
}

// Stop gracefully stops the rotation scheduler.
func (rs *RotationScheduler) Stop() {
	rs.mu.Lock()
	if !rs.running {
		rs.mu.Unlock()
		return
	}
	rs.cancel()
	rs.running = false
	rs.mu.Unlock()

	rs.wg.Wait()
	slog.Info("key rotation scheduler stopped")
}

// run is the main scheduler loop.
func (rs *RotationScheduler) run() {
	defer rs.wg.Done()

	// Run initial check after a short delay
	select {
	case <-time.After(10 * time.Second):
		rs.checkAndRotate()
	case <-rs.ctx.Done():
		return
	}

	ticker := time.NewTicker(rs.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rs.checkAndRotate()
		case <-rs.ctx.Done():
			return
		}
	}
}

// checkAndRotate checks all profiles and rotates expired keys.
func (rs *RotationScheduler) checkAndRotate() {
	if !tpm.IsKeystoreAvailable() {
		return
	}

	profiles, err := tpm.ListKeystoreProfiles()
	if err != nil {
		slog.Error("failed to list keystore profiles", "error", err)
		return
	}

	for _, name := range profiles {
		needs, err := tpm.NeedsRotation(name, rs.maxAge)
		if err != nil {
			slog.Error("failed to check rotation for profile", "profile", name, "error", err)
			continue
		}

		if needs {
			rs.rotateProfile(name)
		}
	}
}

// rotateProfile rotates the encryption keys for a profile.
func (rs *RotationScheduler) rotateProfile(name string) {
	meta, err := tpm.GetProfileMetadata(name)
	if err != nil {
		slog.Error("failed to get profile metadata", "profile", name, "error", err)
		return
	}

	lastRotation := meta.RotatedAt
	if lastRotation.IsZero() {
		lastRotation = meta.CreatedAt
	}

	slog.Info("auto-rotating profile encryption keys",
		"profile", name,
		"version", meta.Version,
		"last_rotation", lastRotation,
		"age", time.Since(lastRotation).Round(time.Hour))

	if err := tpm.RotateProfileKey(name); err != nil {
		slog.Error("failed to auto-rotate profile keys", "profile", name, "error", err)
		return
	}

	slog.Info("profile encryption keys rotated successfully",
		"profile", name,
		"new_version", meta.Version+1)
}
