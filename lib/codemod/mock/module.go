package mock

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/armon/go-radix"
	"github.com/pborman/uuid"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"

	"github.com/qur/withmock/lib/env"
)

type modInfo struct {
	mg        *MockGenerator
	src       string
	path      string
	goVersion string
	fset      *token.FileSet
	deps      map[string]string
	depsTree  *radix.Tree
	mods      map[string]*modInfo
	pkgs      map[string][]*pkgInfo
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
	if mod == "std" {
		mv.Path = "gowm.in/std"
	}
	if err := zip.Unzip(src, mv, zipfile); err != nil {
		return nil, fmt.Errorf("failed to unpack zip %s: %w", zipfile, err)
	}

	mi := &modInfo{
		mg:       m,
		src:      src,
		path:     mod,
		fset:     fset,
		deps:     map[string]string{},
		depsTree: radix.New(),
		mods:     map[string]*modInfo{},
		pkgs:     map[string][]*pkgInfo{},
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
	if f.Go != nil {
		mi.goVersion = f.Go.Version
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
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "internal" || base == "testdata" {
			return fs.SkipDir
		}
		rel, err := filepath.Rel(mi.src, path)
		if err != nil {
			return err
		}
		if err := mi.parsePackage(ctx, path, filepath.Join(mi.path, rel), rel); err != nil {
			return fmt.Errorf("failed to resolve interfaces for %s: %w", rel, err)
		}
		return nil
	})
}

func (mi *modInfo) parsePackage(ctx context.Context, src, path, relPath string) error {
	pkgs, err := parser.ParseDir(mi.fset, src, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if len(pkgs) == 0 {
		// not a go package, ignore it
		return nil
	}
	for name, pkg := range pkgs {
		pi := &pkgInfo{
			mod:      mi,
			name:     name,
			path:     relPath,
			fullPath: path,
			pkg:      pkg,
			files:    map[string]*fileInfo{},
		}
		mi.pkgs[path] = append(mi.pkgs[path], pi)
		if mi.path == "std" {
			// if we are processing the standard library, then add a second
			// entry that points to the same info with the unprefixed name - so
			// that we pull everything from the source we have downloaded
			// instead of using our own standard library.
			mi.pkgs[strings.TrimPrefix(path, "std/")] = append(mi.pkgs[path], pi)
		}
	}
	return nil
}

func (mi *modInfo) resolveAllInterfaces(ctx context.Context) (int, error) {
	count := 0

	for path, pkgs := range mi.pkgs {
		for _, pkg := range pkgs {
			if !strings.HasPrefix(path, mi.path) {
				// ignore external package
				continue
			}
			if err := pkg.resolveInterfaces(ctx); err != nil {
				return 0, fmt.Errorf("failed to resolve interfaces for %s (%s): %w", path, pkg.name, err)
			}
			count += len(pkg.interfaces)
		}
	}

	return count, nil
}

func (mi *modInfo) findPackage(ctx context.Context, path string) ([]*pkgInfo, error) {
	pkg := mi.pkgs[path]
	if pkg != nil {
		return pkg, nil
	}

	if !isModulePath(path) {
		// looks to be a stdlib package
		return mi.findStdlibPackage(ctx, path)
	}

	mod, data, found := mi.depsTree.LongestPrefix(path)
	if !found {
		return nil, fmt.Errorf("unknown package: %s", path)
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

	pkg = info.pkgs[path]
	if pkg == nil {
		return nil, fmt.Errorf("unknown package: %s", path)
	}
	return pkg, nil
}

func (mi *modInfo) findStdlibPackage(ctx context.Context, path string) ([]*pkgInfo, error) {
	env, err := env.GetEnv()
	if err != nil {
		return nil, err
	}
	src := filepath.Join(env["GOROOT"], "src")
	log.Printf("FIND STDLIB: %s in %s", path, src)

	if err := mi.parsePackage(ctx, filepath.Join(src, path), path, ""); err != nil {
		return nil, err
	}

	return mi.pkgs[path], nil
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

func (mi *modInfo) writeModFile(ctx context.Context, dest, mod string) error {
	log.Printf("MODFILE: %s", dest)

	mf := &modfile.File{}
	if err := mf.AddModuleStmt(mod); err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	if mi.goVersion != "" {
		if err := mf.AddGoStmt(mi.goVersion); err != nil {
			return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
		}
	}
	data, err := mf.Format()
	if err != nil {
		return fmt.Errorf("failed to format go.mod for %s: %w", dest, err)
	}

	f, err := os.Create(filepath.Join(dest, "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write go.mod for %s: %w", dest, err)
	}

	buf := &bytes.Buffer{}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dest
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy modfile for %s: %w:\n%s", dest, err, buf.String())
	}
	return nil
}
