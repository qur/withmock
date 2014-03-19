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

func (p *Package) mockFile(base string, m *mockGen) (string, map[string]bool, error) {
	srcFile := filepath.Join(p.src, base)
	filename := filepath.Join(p.dst, base)

	file, err := parser.ParseFile(p.fset, srcFile, nil, parser.ParseComments)
	if err != nil {
		return "", nil, Cerr{"ParseFile", err}
	}

	// If only considering files for this OS/Arch, then reject files
	// that aren't for this OS/Arch based on build constraint (also
	// excludes files with an ignore build constraint).
	if !goodOSArchConstraints(file) {
		return "", nil, nil
	}

	out, err := p.cache.Create(filename)
	if err != nil {
		return "", nil, Cerr{"os.Create", err}
	}
	defer out.Close()

	imports, err := m.file(out, file, srcFile)
	if err != nil {
		return "", nil, Cerr{"m.file", err}
	}

	/*
		// TODO: we want to gofmt, goimports can break things ...
		if err := fixup(filename); err != nil {
			return "", nil, Cerr{"fixup", err}
		}
	*/

	return file.Name.Name, imports, nil
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
		// If only considering files for this OS/Arch, then reject files
		// that aren't for this OS/Arch based on filename.
		if cfg.MatchOSArch && !goodOSArchFile(base, nil) {
			continue
		}

		name, i, err := p.mockFile(base, m)
		if err != nil {
			return "", nil, nil, Cerr{"p.mockFile", err}
		}

		if name == "" {
			continue
		}

		if pkg == "" {
			pkg = name
		} else if name != pkg {
			return "", nil, nil, fmt.Errorf("Package name changed from %s to %s", pkg, name)
		}

		processed++

		for path := range i {
			imports.Set(path, importNormal, "")
		}
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

func (p *Package) mockPackage(byDefault bool, cfg *MockConfig) (_ importSet, ret error) {
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

		w, err := p.cache.Create(output)
		if err != nil {
			return nil, Cerr{"os.Create", err}
		}
		defer func() {
			err := w.Close()
			if ret == nil && err != nil {
				ret = Cerr{"Close", err}
			}
		}()

		if err := rw.Copy(input, w); err != nil {
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
