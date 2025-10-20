package core

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDotGit(t *testing.T) {

	dest, err := unzipHelper("testdata/repofake.zip", t.TempDir())
	require.NoError(t, err)

	g, err := dotGitCheck(filepath.Join(dest, ".git"))
	require.NoError(t, err)

	t.Log(g)
}

func unzipHelper(zipFile, destDir string) (string, error) {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	destPath := filepath.Join(destDir, archive.File[0].Name)

	for _, file := range archive.File {
		filePath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return "", err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}

		fileInArchive, err := file.Open()
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return "", err
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return destPath, nil
}
