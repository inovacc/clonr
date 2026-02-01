package application

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const AppName = "clonr"

var (
	once     sync.Once
	appDir   string
	errorDir error
)

// GetApplicationDirectory returns the clonr configuration directory path.
// Linux: ~/.config/clonr (via os.UserConfigDir)
// Windows: C:\Users\{username}\AppData\Local\clonr (via os.UserCacheDir)
func GetApplicationDirectory() (string, error) {
	once.Do(lazyLoad)

	if errorDir != nil {
		return "", errorDir
	}

	return appDir, errorDir
}

func lazyLoad() {
	var (
		baseDir string
		err     error
	)

	switch runtime.GOOS {
	case "windows":
		// Windows: use AppData\Local (via UserCacheDir)
		baseDir, err = os.UserCacheDir()
	default:
		// Linux/others: use ~/.config (via UserConfigDir)
		baseDir, err = os.UserConfigDir()
	}

	if err != nil {
		errorDir = fmt.Errorf("failed to get config directory: %w", err)
	}

	appDir = filepath.Join(baseDir, AppName)
}
