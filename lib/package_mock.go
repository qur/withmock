// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
)

func getNonGoFiles(path string) ([]string, []string, []string, error) {
	d, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, Cerr{"os.Open", err}
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		return nil, nil, nil, Cerr{"Readdirnames", err}
	}

	nonGoSources := []string{}
	nonGoFiles := []string{}
	subDirs := []string{}

	for _, entry := range files {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			subDirs = append(subDirs, name)
			continue
		}
		if entry.IsDir() || strings.HasSuffix(name, ".go") {
			continue
		}
		if !strings.HasSuffix(name, ".s") && !strings.HasSuffix(name, ".c") {
			nonGoFiles = append(nonGoFiles, name)
			continue
		}
		nonGoSources = append(nonGoSources, name)
	}

	return nonGoSources, nonGoFiles, subDirs, nil
}

func (p *Package) mockFiles(files []string, byDefault bool, cfg *MockConfig, imports importSet) (string, []string, Interfaces, error) {
	interfaces := make(Interfaces)

	pkg := ""

	m := &mockGen{
		fset:           p.fset,
		srcPath:        p.src,
		mockByDefault:  byDefault,
		mockPrototypes: cfg.MockPrototypes,
		callInits:      !cfg.IgnoreInits,
		matchOS:        cfg.MatchOSArch,
		types:          make(map[string]ast.Expr),
		recorders:      make(map[string]string),
		ifInfo:         newIfInfo(""),
		MOCK:           cfg.MOCK,
		EXPECT:         cfg.EXPECT,
		ObjEXPECT:      cfg.ObjEXPECT,
	}

	m.ifInfo.EXPECT = m.EXPECT

	processed := 0

	cfg.MatchOSArch = true

	for _, base := range files {
		srcFile := filepath.Join(p.src, base)
		filename := filepath.Join(p.dst, base)

		// If only considering files for this OS/Arch, then reject files
		// that aren't for this OS/Arch based on filename.
		if cfg.MatchOSArch && !goodOSArchFile(base, nil) {
			continue
		}

		file, err := parser.ParseFile(p.fset, srcFile, nil, parser.ParseComments)
		if err != nil {
			return "", nil, nil, Cerr{"ParseFile", err}
		}

		// If only considering files for this OS/Arch, then reject files
		// that aren't for this OS/Arch based on build constraint (also
		// excludes files with an ignore build constraint).
		if cfg.MatchOSArch && !goodOSArchConstraints(file) {
			continue
		}

		if pkg == "" {
			pkg = file.Name.Name
		} else if file.Name.Name != pkg {
			return "", nil, nil, fmt.Errorf("Package name changed from %s to %s", pkg, file.Name.Name)
		}

		processed++

		out, err := os.Create(filename)
		if err != nil {
			return "", nil, nil, Cerr{"os.Create", err}
		}
		defer out.Close()

		i, err := m.file(out, file, srcFile)
		if err != nil {
			return "", nil, nil, Cerr{"m.file", err}
		}

		for path := range i {
			imports.Set(path, importNormal, "")
		}

		/*
			// TODO: we want to gofmt, goimports can break things ...
			err = fixup(filename)
			if err != nil {
				return err
			}
		*/
	}

	// If we skipped over all the files for this package, then ignore it
	// entirely.
	if processed == 0 {
		return "", nil, nil, nil
	}

	filename := filepath.Join(p.dst, pkg+"_mock.go")

	out, err := os.Create(filename)
	if err != nil {
		return "", nil, nil, Cerr{"os.Create", err}
	}
	defer out.Close()

	err = m.pkg(out, pkg)
	if err != nil {
		return "", nil, nil, Cerr{"m.pkg", err}
	}

	// TODO: currently we need to use goimports to add missing imports, we
	// need to sort out our own imports, then we can switch to gofmt.
	err = fixup(filename)
	if err != nil {
		return "", nil, nil, Cerr{"fixup", err}
	}

	m.ifInfo.filename = filepath.Join(p.dst, pkg+"_ifmocks.go")
	interfaces[pkg] = m.ifInfo

	return pkg, m.extFunctions, interfaces, nil
}

func (p *Package) mockPackage(byDefault bool, cfg *MockConfig) (importSet, error) {
	imports := make(importSet)

	processDir := func(path, rel string) error {
		if path == p.src {
			return nil
		}

		imports.Set(filepath.Join(p.name, rel), importNormal, "")
		return filepath.SkipDir
	}

	goFiles, nonGoFiles, nonGoSources := []string{}, []string{}, []string{}

	processFile := func(path, rel string) error {
		name := filepath.Base(path)
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if strings.HasSuffix(name, ".go") {
			goFiles = append(goFiles, name)
			return nil
		}
		if !strings.HasSuffix(name, ".s") && !strings.HasSuffix(name, ".c") {
			nonGoFiles = append(nonGoFiles, name)
			return nil
		}
		nonGoSources = append(nonGoSources, name)
		return nil
	}

	if err := walk(p.src, p.dst, processDir, processFile); err != nil {
		return nil, Cerr{"walk", err}
	}

	pkg, externalFunctions, interfaces, err := p.mockFiles(goFiles, byDefault, cfg, imports)
	if err != nil {
		return nil, Cerr{"mockFiles", err}
	}

	if err := genInterfaces(interfaces); err != nil {
		return nil, Cerr{"genInterfaces", err}
	}

	if cfg.IgnoreNonGoFiles {
		return imports, nil
	}

	// Load up a rewriter with the rewrites for the external functions
	rw := NewRewriter(nil)
	for _, fname := range externalFunctions {
		rw.Rewrite("·" + fname + "(", "·_real_" + fname + "(")
		if p.rw != nil {
			p.rw.Rewrite(pkg + "·" + fname + "(", pkg + "·_real_" + fname + "(")
		}
	}

	// Now copy the non go source files through the rewriter
	for _, name := range nonGoSources {
		input := filepath.Join(p.src, name)
		output := filepath.Join(p.dst, name)

		err := rw.Copy(input, output)
		if err != nil {
			return nil, Cerr{"rw.Copy", err}
		}
	}

	// Symlink non source files
	for _, name := range nonGoFiles {
		input := filepath.Join(p.src, name)
		output := filepath.Join(p.dst, name)

		err := os.Symlink(input, output)
		if err != nil {
			return nil, Cerr{"os.Symlink", err}
		}
	}

	return imports, nil
}
