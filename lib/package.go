// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"log"
)

type Package struct {
	name string
	label string
	install bool
	src, dst string
	pkgDst string
	tmpDir string
	tmpPath string
	goPath string
	rw *rewriter
	fset *token.FileSet
	cache *Cache
	files []string
}

func NewPackage(pkgName, label, tmpDir, goPath string) (*Package, error) {
	codeSrc, err := LookupImportPath(pkgName)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}

	cache, err := OpenCache()
	if err != nil {
		return nil, Cerr{"OpenCache", err}
	}

	tmpPath := getTmpPath(tmpDir)

	return &Package{
		name: pkgName,
		label: label,
		install: true,
		src: codeSrc,
		dst: filepath.Join(tmpPath, "src", label),
		pkgDst: filepath.Join(tmpPath, "pkg", GetOsArch(), label + ".a"),
		tmpDir: tmpDir,
		tmpPath: tmpPath,
		goPath: goPath,
		rw: nil,
		fset: token.NewFileSet(),
		cache: cache,
	}, nil
}

func NewStdlibPackage(pkgName, label, tmpDir, goRoot string, rw *rewriter) (*Package, error) {
	log.Printf("START: lookup import path")
	codeSrc, err := LookupImportPath(pkgName)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}
	log.Printf("END: lookup import path")

	log.Printf("START: open cache")
	cache, err := OpenCache()
	if err != nil {
		return nil, Cerr{"OpenCache", err}
	}
	log.Printf("END: open cache")

	tmpPath := getTmpPath(tmpDir)
	tmpRoot := getTmpRoot(tmpDir)

	return &Package{
		name: pkgName,
		label: label,
		install: true,
		src: codeSrc,
		dst: filepath.Join(tmpRoot, "src", "pkg", label),
		pkgDst: filepath.Join(tmpRoot, "pkg", GetOsArch(), label + ".a"),
		tmpDir: tmpDir,
		tmpPath: tmpPath,
		goPath: goRoot,
		rw: rw,
		fset: token.NewFileSet(),
		cache: cache,
	}, nil
}

func (p *Package) Name() string {
	return p.name
}

func (p *Package) Label() string {
	return p.label
}

func (p *Package) Loc() codeLoc {
	return codeLoc{p.src, p.dst}
}

func (p *Package) getImports(tests bool) (importSet, error) {
	imports := make(importSet)

	isGoFile := func(info os.FileInfo) bool {
		if info.IsDir() {
			return false
		}
		if !tests && strings.HasSuffix(info.Name(), "_test.go") {
			return false
		}
		return strings.HasSuffix(info.Name(), ".go")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, p.src, isGoFile,
		parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, i := range file.Imports {
				path := strings.Trim(i.Path.Value, "\"")
				comment := strings.TrimSpace(i.Comment.Text())

				if strings.HasPrefix(path, "_mock_/") {
					path = path[7:]
					comment = "mock"
				}

				mode := importNormal
				path2 := ""
				switch {
				case strings.ToLower(comment) == "mock":
					mode = importMock
				case strings.HasPrefix(comment, "replace("):
					mode = importReplace
					path2 = comment[8:len(comment)-1]
				}

				err := imports.Set(path, mode, path2)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return imports, nil
}

func (p *Package) GetImports() (importSet, error) {
	return p.getImports(true)
}

func (p *Package) MockImports(importNames map[string]string, cfg *Config) error {
	return processSingleDir(p.src, p.dst, func(path, rel string) error {
		target := filepath.Join(p.dst, rel)

		p.files = append(p.files, path)

		// Non-code we leave alone, code may need modification
		if !strings.HasSuffix(path, ".go") {
			return os.Symlink(path, target)
		}

		return mockFileImports(path, target, importNames, cfg)
	})
}

func (p *Package) symlinkFile(path, rel string) error {
	target := filepath.Join(p.dst, rel)

	p.files = append(p.files, path)

	return os.Symlink(path, target)
}

func (p *Package) rewriteFile(path, rel string) error {
	target := filepath.Join(p.dst, rel)

	p.files = append(p.files, path)

	// TODO: the rewrites need to be part of the key ...

	key := p.cache.NewCacheFileKey("rewriteFile", path)
	w, err := p.cache.GetFile(key)
	if err != nil {
		return Cerr{"os.Create", err}
	}
	defer w.Close()

	if !w.HasData() {
		if err := p.rw.Copy(path, w); err != nil {
			return Cerr{"p.rw.Copy", err}
		}
	}

	return w.Install(target)
}

func (p *Package) disableFile(path, rel string) error {
	target := filepath.Join(p.dst, rel)

	p.files = append(p.files, path)

	if strings.HasSuffix(path, ".go") {
		return addMockDisables(path, target)
	}

	return os.Symlink(path, target)
}

func (p *Package) Link() (importSet, error) {
	if err := processTree(p.src, p.dst, p.symlinkFile); err != nil {
		return nil, Cerr{"processTree", err}
	}

	return p.getImports(false)
}

func (p *Package) DisableInstall() {
	p.install = false
}

func (p *Package) Replace(with string) (importSet, error) {
	src, err := LookupImportPath(with)
	if err != nil {
		return nil, Cerr{"LookupImportPath", err}
	}

	if err := processTree(src, p.dst, p.symlinkFile); err != nil {
		return nil, Cerr{"processTree", err}
	}

	return p.getImports(false)
}

func (p *Package) Rewrite() (importSet, error) {
	if err := processTree(p.src, p.dst, p.rewriteFile); err != nil {
		return nil, Cerr{"processTree", err}
	}

	return p.getImports(false)
}

func (p *Package) DisableAllMocks() error {
	return processTree(p.src, p.dst, p.disableFile)
}

func (p *Package) Gen(mock bool, cfg *MockConfig) (importSet, error) {
	log.Printf("PKG GEN: %s", p.name)

	if err := os.MkdirAll(p.dst, 0700); err != nil {
		return nil, Cerr{"os.MkdirAll", err}
	}

	return p.mockPackage(mock, cfg)
}

func (p *Package) Deps() ([]string, error) {
	deps := []string{}

	err := processSingleDir(p.src, p.dst, func(path, _ string) error {
		p.files = append(p.files, path)

		if strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			// Don't try and parse imports from non-go or test files.
			return nil
		}

		f, err := parser.ParseFile(p.fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return Cerr{"ParseFile("+path+")", err}
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, "\"")
			if impPath == "C" {
				continue
			}
			deps = append(deps, impPath)
		}

		return nil
	})
	if err != nil {
		return nil, Cerr{"processSingleDir", err}
	}

	return deps, nil
}

func (p *Package) insideCommand(command string, args ...string) *exec.Cmd {
	env := os.Environ()

	// remove any current GOPATH from the environment
	for i := range env {
		if strings.HasPrefix(env[i], "GOPATH=") {
			env[i] = "__IGNORE="
		}
		if strings.HasPrefix(env[i], "GOROOT=") {
			env[i] = "__IGNORE="
		}
	}

	// Setup the environment variables that we want
	env = append(env, "GOPATH=" + p.tmpPath)
	env = append(env, "GOROOT=" + getTmpRoot(p.tmpDir))
	//env = append(env, "ORIG_GOPATH=" + c.origPath)

	cmd := exec.Command(command, args...)
	cmd.Env = env
	return cmd
}

func (p *Package) needsInstall() (bool, error) {
	if !p.install {
		return false, nil
	}

	if getMark(p.label) == testMark {
		// we don't install packages marked for test
		return false, nil
	}

	d, err := os.Open(p.dst)
	if err != nil {
		return false, Cerr{"os.Open", err}
	}
	defer d.Close()

	files, err := d.Readdirnames(-1)
	if err != nil {
		return false, Cerr{"d.Readdirnames", err}
	}

	for _, name := range files {
		if strings.HasSuffix(name, ".go") {
			return true, nil
		}
	}

	return false, nil
}

func (p *Package) Install() error {
	needsInstall, err := p.needsInstall()
	if err != nil {
		return Cerr{"p.needsInstall", err}
	}

	if !needsInstall {
		return nil
	}

	key := p.cache.NewCacheFileKey("install", p.files...)
	f, err := p.cache.GetFile(key)
	if err != nil {
		return Cerr{"cache.GetFile", err}
	}
	defer f.Close()

	if !f.HasData() {
		err := f.WriteFunc(func(dest string) error {
			cmd := p.insideCommand("go", "build", "-v", "-o", dest, p.label)
			out, err := cmd.CombinedOutput()
			log.Printf("go build %s:\n%s", p.label, out)
			if err != nil {
				return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
					p.label, err, out)
			}
			return nil
		})
		if err != nil {
			return Cerr{"WriteFunc", err}
		}
	}

	if err := f.Install(p.pkgDst); err != nil {
		return Cerr{"f.Install", err}
	}

	return nil
}
