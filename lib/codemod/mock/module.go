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

	"github.com/armon/go-radix"
	"github.com/pborman/uuid"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

type modInfo struct {
	mg       *MockGenerator
	src      string
	fset     *token.FileSet
	deps     map[string]string
	depsTree *radix.Tree
	mods     map[string]*modInfo
	pkgs     map[string]*pkgInfo
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
		mg:       m,
		src:      src,
		fset:     fset,
		deps:     map[string]string{},
		depsTree: radix.New(),
		mods:     map[string]*modInfo{},
		pkgs:     map[string]*pkgInfo{},
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
		mi.depsTree.Insert(req.Mod.Path, req.Mod.Version[1:])
	}

	if err := mi.discoverPackages(ctx); err != nil {
		return nil, err
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

func (mi *modInfo) discoverPackages(ctx context.Context) error {
	return filepath.WalkDir(mi.src, func(path string, d fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		if err != nil || !d.IsDir() {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") || filepath.Base(path) == "internal" {
			return fs.SkipDir
		}
		rel, err := filepath.Rel(mi.src, path)
		if err != nil {
			return err
		}
		if err := mi.parsePackage(ctx, rel); err != nil {
			return fmt.Errorf("failed to resolve interfaces for %s: %w", rel, err)
		}
		return nil
	})
}

func (mi *modInfo) parsePackage(ctx context.Context, path string) error {
	pkgs, err := parser.ParseDir(mi.fset, filepath.Join(mi.src, path), nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	names := []string{}
	for name := range pkgs {
		if strings.HasSuffix(name, "_test") {
			continue
		}
		names = append(names, name)
	}
	switch len(names) {
	case 0:
		// not a go package, ignore it
		return nil
	case 1:
		// this is what we want
	default:
		return fmt.Errorf("don't know how to handle more than one package (got %s)", names)
	}
	name := names[0]
	pi := &pkgInfo{
		mod:        mi,
		name:       name,
		pkg:        pkgs[name],
		files:      map[string]*fileInfo{},
		interfaces: map[string]*interfaceInfo{},
	}
	mi.pkgs[path] = pi
	return nil
}

func (mi *modInfo) resolveAllInterfaces(ctx context.Context) (int, error) {
	count := 0

	for path, pkg := range mi.pkgs {
		if err := pkg.resolveInterfaces(ctx); err != nil {
			return 0, fmt.Errorf("failed to process %s (%s): %w", path, pkg.name, err)
		}
		count += len(pkg.interfaces)
	}

	return count, nil
}

func (mi *modInfo) findPackage(ctx context.Context, fullPath string) (*pkgInfo, error) {
	if !isModulePath(fullPath) {
		// looks to be a stdlib package
		return mi.findStdlibPackage(ctx, fullPath)
	}

	mod, data, found := mi.depsTree.LongestPrefix(fullPath)
	if !found {
		return nil, fmt.Errorf("unknown package: %s", fullPath)
	}
	ver := data.(string)
	info := mi.mods[mod]
	if info == nil {
		i, err := mi.mg.getModInfo(ctx, mi.fset, mod, ver)
		if err != nil {
			return nil, err
		}
		mi.mods[mod] = i
		info = i
	}
	rel, err := filepath.Rel(mod, fullPath)
	if err != nil {
		return nil, err
	}
	pkg := mi.pkgs[rel]
	if pkg == nil {
		return nil, fmt.Errorf("unknown package: %s", fullPath)
	}
	return pkg, nil
}

func (mi *modInfo) findStdlibPackage(ctx context.Context, fullPath string) (*pkgInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func isModulePath(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false
	}
	if strings.Contains(parts[0], ".") {
		return true
	}
	return false
}
