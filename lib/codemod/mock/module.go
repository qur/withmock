package mock

import (
	"context"
	"fmt"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

type modInfo struct {
	src  string
	fset *token.FileSet
	deps map[string]string
	pkgs map[string]*pkgInfo
}

func (m *MockGenerator) getModInfo(ctx context.Context, fset *token.FileSet, mod, ver string) (*modInfo, error) {
	zipfile, err := m.downloadModule(ctx, mod, ver)
	if err != nil {
		return nil, err
	}

	src := filepath.Join(m.scratch, mod, ver, "src")
	if err := os.MkdirAll(src, 0755); err != nil {
		return nil, err
	}

	mv := module.Version{Path: mod, Version: "v" + ver}
	if err := zip.Unzip(src, mv, zipfile); err != nil {
		return nil, fmt.Errorf("failed to unpack zip %s: %w", zipfile, err)
	}

	mi := &modInfo{
		src:  src,
		fset: fset,
		deps: map[string]string{},
	}

	modFile := filepath.Join(src, "go.mod")
	modData, err := os.ReadFile(modFile)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse(modFile, modData, nil)
	if err != nil {
		return nil, err
	}
	for _, req := range f.Require {
		if req.Indirect {
			continue
		}
		log.Printf("REQUIRE: %s", req.Mod)
		mi.deps[req.Mod.Path] = req.Mod.Version[1:]
	}

	return mi, nil
}

func (m *MockGenerator) downloadModule(ctx context.Context, mod, ver string) (string, error) {
	src, err := m.s.Source(ctx, mod, ver)
	if err != nil {
		return "", err
	}
	if rc, ok := src.(io.ReadCloser); ok {
		defer rc.Close()
	}

	path := filepath.Join(m.scratch, mod, ver, "src.zip")

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		return "", err
	}

	return path, nil
}

func (mi *modInfo) resolveAllInterfaces(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("not implemented")
}
