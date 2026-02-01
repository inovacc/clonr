package params

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/inovacc/clonr/internal/application"
)

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

	AppdataDir = filepath.Join(dir, application.AppName)

	if err := os.MkdirAll(AppdataDir, os.ModePerm); err != nil {
		panic(err)
	}
}
