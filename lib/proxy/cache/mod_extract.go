package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pborman/uuid"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

func extractModFile(r io.Reader, scratch, mod, ver string) (io.Reader, error) {
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}

	d := filepath.Join(scratch, mod, ver, uuid.New())

	src := filepath.Join(d, "src")
	zipFile := filepath.Join(d, "src.zip")
	modFile := filepath.Join(src, "go.mod")

	if err := os.MkdirAll(src, 0755); err != nil {
		return nil, fmt.Errorf("failed to create modExtractor temp (%s, %s): %w", mod, ver, err)
	}

	zf, err := os.Create(zipFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create modExtractor zip file (%s, %s): %w", mod, ver, err)
	}
	defer zf.Close()

	if _, err := io.Copy(zf, r); err != nil {
		return nil, fmt.Errorf("failed to write modExtractor zip file (%s, %s): %w", mod, ver, err)
	}

	if err := zip.Unzip(src, module.Version{Path: mod, Version: "v" + ver}, zipFile); err != nil {
		return nil, fmt.Errorf("failed to extract modExtractor zip file (%s, %s): %w", mod, ver, err)
	}

	f, err := os.Open(modFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open modExtractor mod file (%s, %s): %w", mod, ver, err)
	}

	if err := os.RemoveAll(src); err != nil {
		return nil, fmt.Errorf("failed to cleanup modExtractor (%s, %s): %w", mod, ver, err)
	}

	return f, err
}
