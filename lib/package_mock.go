// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"go/ast"
	"go/parser"
	"go/token"
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

// MakePkg writes a mock version of the package found at p.src into p.dst.
// If p.dst already exists, bad things will probably happen.
func (p *Package) makePkg(mock bool, cfg *MockConfig) (importSet, error) {
	isGoFile := func(info os.FileInfo) bool {
		if info.IsDir() {
			return false
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return false
		}
		return strings.HasSuffix(info.Name(), ".go")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, p.src, isGoFile, parser.ParseComments)
	if err != nil {
		return nil, Cerr{"parseDir", err}
	}

	imports := make(importSet)

	nonGoSources, nonGoFiles, subDirs, err := getNonGoFiles(p.src)
	if err != nil {
		return nil, Cerr{"getNonGoFiles", err}
	}

	for _, name := range subDirs {
		imports.Set(filepath.Join(p.name, name), importNormal, "")
	}

	externalFunctions := map[string][]string{}

	interfaces := make(Interfaces)

	for name, pkg := range pkgs {
		m := &mockGen{
			fset:           fset,
			srcPath:        p.src,
			mockByDefault:  mock,
			mockPrototypes: cfg.MockPrototypes,
			callInits:      !cfg.IgnoreInits,
			matchOS:        cfg.MatchOSArch,
			types:          make(map[string]ast.Expr),
			recorders:      make(map[string]string),
			ifInfo:         newIfInfo(filepath.Join(p.dst, name+"_ifmocks.go")),
			MOCK:           cfg.MOCK,
			EXPECT:         cfg.EXPECT,
			ObjEXPECT:      cfg.ObjEXPECT,
		}

		m.ifInfo.EXPECT = m.EXPECT

		processed := 0

		cfg.MatchOSArch = true

		for path, file := range pkg.Files {
			base := filepath.Base(path)

			srcFile := filepath.Join(p.src, base)
			filename := filepath.Join(p.dst, base)

			// If only considering files for this OS/Arch, then reject files
			// that aren't for this OS/Arch based on filename.
			if cfg.MatchOSArch && !goodOSArchFile(base, nil) {
				continue
			}

			// If only considering files for this OS/Arch, then reject files
			// that aren't for this OS/Arch based on build constraint (also
			// excludes files with an ignore build constraint).
			if cfg.MatchOSArch && !goodOSArchConstraints(file) {
				continue
			}

			processed++

			out, err := os.Create(filename)
			if err != nil {
				return nil, Cerr{"os.Create", err}
			}
			defer out.Close()

			i, err := m.file(out, file, srcFile)
			if err != nil {
				return nil, Cerr{"m.file", err}
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
			continue
		}

		filename := filepath.Join(p.dst, name+"_mock.go")

		out, err := os.Create(filename)
		if err != nil {
			return nil, Cerr{"os.Create", err}
		}
		defer out.Close()

		err = m.pkg(out, name)
		if err != nil {
			return nil, Cerr{"m.pkg", err}
		}

		// TODO: currently we need to use goimports to add missing imports, we
		// need to sort out our own imports, then we can switch to gofmt.
		err = fixup(filename)
		if err != nil {
			return nil, Cerr{"fixup", err}
		}

		externalFunctions[name] = m.extFunctions

		interfaces[name] = m.ifInfo
	}

	if err := genInterfaces(interfaces); err != nil {
		return nil, Cerr{"genInterfaces", err}
	}

	if cfg.IgnoreNonGoFiles {
		return imports, nil
	}

	// Load up a rewriter with the rewrites for the external functions
	rw := NewRewriter(nil)
	for pkg, funcs := range externalFunctions {
		for _, name := range funcs {
			rw.Rewrite("·" + name + "(", "·_real_" + name + "(")
			if p.rw != nil {
				p.rw.Rewrite(pkg + "·" + name + "(", pkg + "·_real_" + name + "(")
			}
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
