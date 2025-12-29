package params

import (
	"os"
	"path/filepath"
	"sync"
)

const appName = "clonr"

var (
	once       sync.Once
	AppdataDir string
)

func init() {
	once.Do(getAppDataDir)
}

func getAppDataDir() {
	dir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	AppdataDir = filepath.Join(dir, appName)

	if err := os.MkdirAll(AppdataDir, os.ModePerm); err != nil {
		panic(err)
	}
}
