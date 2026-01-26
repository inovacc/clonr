package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/inovacc/autoupdater"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateForce     bool
	updatePreRel    bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	Long: `Check for new versions of clonr and optionally install them.

The update command connects to GitHub Releases to check for newer versions.
If an update is available, it will download, verify (SHA256), and install it.

Examples:
  clonr update              # Check and install if available
  clonr update --check      # Only check, don't install
  clonr update --force      # Reinstall current version
  clonr update --pre        # Include pre-release versions`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVarP(&updateCheckOnly, "check", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Force reinstall of current version")
	updateCmd.Flags().BoolVar(&updatePreRel, "pre", false, "Include pre-release versions")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		cancel()
	}()

	// Create updater with progress hooks
	updater, err := autoupdater.New(autoupdater.Config{
		Owner:          "inovacc",
		Repo:           "clonr",
		CurrentVersion: Version,
		PreRelease:     updatePreRel,
		Hooks:          &updateHooks{},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	// Check for updates
	release, err := updater.Check(ctx)
	if err != nil {
		if errors.Is(err, autoupdater.ErrNoUpdateAvailable) {
			if updateForce {
				fmt.Println("No newer version available, but --force specified.")
				fmt.Println("Reinstalling current version...")
				return forceReinstall(ctx)
			}
			fmt.Println("Already up to date!")
			return nil
		}
		if errors.Is(err, autoupdater.ErrAssetNotFound) {
			return fmt.Errorf("no release asset found for your platform")
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Printf("\nUpdate available: %s -> %s\n", Version, release.Version)

	if release.IsGoReleaser {
		fmt.Println("Release type: GoReleaser")
	}

	if updateCheckOnly {
		fmt.Println("\nUse 'clonr update' without --check to install.")
		return nil
	}

	// Apply update
	fmt.Println("\nDownloading and installing...")
	if err := updater.Apply(ctx, release); err != nil {
		if errors.Is(err, autoupdater.ErrVerificationFailed) {
			return fmt.Errorf("checksum verification failed - download may be corrupted")
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}

	return nil
}

func forceReinstall(ctx context.Context) error {
	// For force reinstall, we create a new updater that accepts any version
	forceUpdater, err := autoupdater.New(autoupdater.Config{
		Owner:          "inovacc",
		Repo:           "clonr",
		CurrentVersion: "0.0.0", // Pretend we're on an old version
		Hooks:          &updateHooks{},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	release, err := forceUpdater.Check(ctx)
	if err != nil {
		return fmt.Errorf("failed to find release: %w", err)
	}

	fmt.Printf("Reinstalling version %s...\n", release.Version)
	if err := forceUpdater.Apply(ctx, release); err != nil {
		return fmt.Errorf("failed to reinstall: %w", err)
	}

	return nil
}

// updateHooks implements autoupdater.UpdateHooks for progress display
type updateHooks struct {
	lastPercent int
}

func (h *updateHooks) OnUpdateAvailable(current, latest string) bool {
	// Auto-accept updates (user already confirmed by running the command)
	return true
}

func (h *updateHooks) OnBeforeUpdate(ctx context.Context) error {
	// Nothing to save for clonr
	return nil
}

func (h *updateHooks) OnProgress(downloaded, total int64) {
	if total > 0 {
		percent := int(float64(downloaded) / float64(total) * 100)
		if percent != h.lastPercent {
			h.lastPercent = percent
			_, _ = fmt.Printf("\rDownloading: %3d%% (%d/%d bytes)", percent, downloaded, total)
		}
	}
}

func (h *updateHooks) OnUpdateComplete(newVersion string) {
	_, _ = fmt.Printf("\n\nSuccessfully updated to %s!\n", newVersion)
	fmt.Println("Please restart clonr to use the new version.")
}

func (h *updateHooks) OnUpdateError(err error) {
	_, _ = fmt.Printf("\nUpdate failed: %v\n", err)
}
