package mock

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pborman/uuid"
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
	path := filepath.Join(m.scratch, mod, ver, uuid.New())
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	zipfile := filepath.Join(path, "src.zip")
	if err := m.downloadModule(ctx, mod, ver, zipfile); err != nil {
		return nil, err
	}

	src := filepath.Join(path, "src")
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

func (m *MockGenerator) downloadModule(ctx context.Context, mod, ver, dest string) error {
	src, err := m.s.Source(ctx, mod, ver)
	if err != nil {
		return err
	}
	if rc, ok := src.(io.ReadCloser); ok {
		defer rc.Close()
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		return err
	}

	return nil
}

func (mi *modInfo) resolveAllInterfaces(ctx context.Context) (int, error) {
	count := 0

	if err := filepath.WalkDir(mi.src, func(path string, d fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		if err != nil || !d.IsDir() {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return fs.SkipDir
		}
		rel, err := filepath.Rel(mi.src, path)
		if err != nil {
			return err
		}
		n, err := mi.resolveInterfaces(ctx, rel)
		if err != nil {
			return fmt.Errorf("failed to resolve interfaces for %s: %w", rel, err)
		}
		count += n
		return nil
	}); err != nil {
		return 0, err
	}

	return count, nil
}

func (mi *modInfo) resolveInterfaces(ctx context.Context, path string) (int, error) {
	log.Printf("RESOLVE INTERFACES: %s", path)
	pkgs, err := parser.ParseDir(mi.fset, filepath.Join(mi.src, path), nil, parser.ParseComments)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	names := []string{}
	for name := range pkgs {
		if strings.HasSuffix(name, "_test") {
			continue
		}
		names = append(names, name)
	}
	if len(names) != 1 {
		return 0, fmt.Errorf("don't know how to handle more than one package (got %s)", names)
	}
	name := names[0]
	pi := &pkgInfo{
		mod:  mi,
		name: name,
	}
	n, err := pi.resolveInterfaces(ctx, pkgs[name])
	if err != nil {
		return 0, fmt.Errorf("failed to process %s (%s): %w", path, name, err)
	}
	mi.pkgs[path] = pi
	return n, nil
}
