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

	"github.com/qur/withmock/cache"
	"github.com/qur/withmock/config"
	"github.com/qur/withmock/utils"
)

func (p *Package) mockFile(base string, m *mockGen) (string, map[string]bool, error) {
	srcFile := filepath.Join(p.src, base)
	filename := filepath.Join(p.dst, base)

	key, err := p.cache.NewCacheFileKey("mockFile", srcFile)
	if err != nil {
		return "", nil, utils.Err{"cache.NewCacheFileKey", err}
	}

	f, err := p.cache.GetFile(key, filename)
	if err != nil {
		return "", nil, utils.Err{"cache.GetFile", err}
	}
	defer f.Close()

	var (
		name string
		info *mockFileInfo
		ignore bool
		file *ast.File
	)

	if f.Has("ignore") {
		ignore = f.Get("ignore").(bool)
	} else {
		var err error

		file, err = parser.ParseFile(p.fset, srcFile, nil, parser.ParseComments)
		if err != nil {
			return "", nil, utils.Err{"ParseFile", err}
		}

		// If only considering files for this OS/Arch, then reject files
		// that aren't for this OS/Arch based on build constraint (also
		// excludes files with an ignore build constraint).
		ignore = !goodOSArchConstraints(file)

		f.Store("ignore", ignore)
	}

	if ignore {
		if err := f.Save(); err != nil {
			return "", nil, utils.Err{"f.Save", err}
		}

		return "", nil, nil
	}

	if f.Has(cache.Data, "name", "info") {
		name = f.Get("name").(string)
		info = f.Get("info").(*mockFileInfo)
	} else {
		if file == nil {
			var err error

			file, err = parser.ParseFile(p.fset, srcFile, nil, parser.ParseComments)
			if err != nil {
				return "", nil, utils.Err{"ParseFile", err}
			}
		}

		i, err := m.file(f, file, srcFile)
		if err != nil {
			return "", nil, utils.Err{"m.file", err}
		}

		f.Store("name", file.Name.Name)
		f.Store("info", i)

		name = file.Name.Name
		info = i
	}

	// Update m using the information provided by m.file

	for n, t := range info.Types {
		m.types[n] = t
		m.ifInfo.addType(n, t, info.ImportMap)
		breakLoops(t)
	}

	for t, r := range info.Recorders {
		m.recorders[t] = r
	}

	for _, name := range info.ExtFunctions {
		m.extFunctions = append(m.extFunctions, name)
	}

	// Create the actual file in the src tree
	if err := f.Install(); err != nil {
		return "", nil, utils.Err{"f.Install", err}
	}

	/*
		// TODO: we want to gofmt, goimports can break things ...
		if err := fixup(filename); err != nil {
			return "", nil, utils.Err{"fixup", err}
		}
	*/

	// Construct the imports map to return

	imports := map[string]bool{
		"github.com/qur/gomock/interfaces": false,
	}

	for _, impPath := range info.ImportMap {
		imports[impPath] = false
	}

	return name, imports, nil
}

func (p *Package) mockFiles(files []string, byDefault bool, cfg *config.MockConfig, imports importSet) (string, []string, Interfaces, error) {
	interfaces := make(Interfaces)

	pkg := ""

	m := &mockGen{
		fset:           p.fset,
		srcPath:        p.src,
		mockByDefault:  byDefault,
		types:          make(map[string]ast.Expr),
		recorders:      make(map[string]string),
		ifInfo:         newIfInfo(""),
		MOCK:           cfg.MOCK,
		EXPECT:         cfg.EXPECT,
		ObjEXPECT:      cfg.ObjEXPECT,
	}

	m.ifInfo.EXPECT = m.EXPECT

	processed := 0

	for _, base := range files {
		// Reject files that aren't for this OS/Arch based on filename.
		if !goodOSArchFile(base, nil) {
			continue
		}

		name, i, err := p.mockFile(base, m)
		if err != nil {
			return "", nil, nil, utils.Err{"p.mockFile", err}
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

	key, err := p.cache.NewCacheFileKey("mockFiles.pkg", p.files...)
	if err != nil {
		return "", nil, nil, utils.Err{"cache.NewCacheFileKey", err}
	}

	f, err := p.cache.GetFile(key, filename)
	if err != nil {
		return "", nil, nil, utils.Err{"cache.GetFile", err}
	}
	defer f.Close()

	if !f.HasData() {
		err := f.WriteFunc(func(filename string) error {
			if err := p.doPkg(filename, pkg, m); err != nil {
				return utils.Err{"p.doPkg", err}
			}
			return nil
		})
		if err != nil {
			return "", nil, nil, utils.Err{"f.WriteFunc", err}
		}
	}

	if err := f.Install(); err != nil {
		return "", nil, nil, utils.Err{"f.Install", err}
	}

	m.ifInfo.filename = filepath.Join(p.dst, pkg+"_ifmocks.go")
	interfaces[pkg] = m.ifInfo

	return pkg, m.extFunctions, interfaces, nil
}

func (p *Package) doPkg(filename, pkg string, m *mockGen) error {
	out, err := os.Create(filename)
	if err != nil {
		return utils.Err{"os.Create", err}
	}
	defer out.Close()

	err = m.pkg(out, pkg)
	if err != nil {
		return utils.Err{"m.pkg", err}
	}

	// TODO: currently we need to use goimports to add missing imports, we
	// need to sort out our own imports, then we can switch to gofmt.
	err = fixup(filename)
	if err != nil {
		return utils.Err{"fixup", err}
	}

	return nil
}

func (p *Package) mockPackage(byDefault bool, cfg *config.MockConfig) (importSet, error) {
	imports := make(importSet)

	processDir := func(path, rel string) error {
		if path == p.src {
			return nil
		}

		imports.Set(filepath.Join(p.name, rel), importNoInstall, "")
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

		p.files = append(p.files, path)

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

	if err := walk(p.src, processDir, processFile); err != nil {
		return nil, utils.Err{"walk", err}
	}

	pkg, externalFunctions, interfaces, err := p.mockFiles(goFiles, byDefault, cfg, imports)
	if err != nil {
		return nil, utils.Err{"mockFiles", err}
	}

	if err := p.genInterfaces(interfaces); err != nil {
		return nil, utils.Err{"genInterfaces", err}
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

		key, err := p.cache.NewCacheFileKey("mockPackage.nonGoSource", input)
		if err != nil {
			return nil, utils.Err{"cache.NewCacheFileKey", err}
		}

		w, err := p.cache.GetFile(key, output)
		if err != nil {
			return nil, utils.Err{"os.Create", err}
		}
		defer w.Close()

		if err := rw.Copy(input, w); err != nil {
			return nil, utils.Err{"rw.Copy", err}
		}

		if err := w.Install(); err != nil {
			return nil, utils.Err{"w.Install", err}
		}
	}

	// Symlink non source files
	for _, name := range nonGoFiles {
		input := filepath.Join(p.src, name)
		output := filepath.Join(p.dst, name)

		err := os.Symlink(input, output)
		if err != nil {
			return nil, utils.Err{"os.Symlink", err}
		}
	}

	return imports, nil
}

func (p *Package) genInterfaces(interfaces Interfaces) error {
	for name, i := range interfaces {
		if i.filename == "" {
			// no filename means this package was only parsed for information,
			// we don't need to write anything out
			continue
		}

		key, err := p.cache.NewCacheFileKey("genInterface."+name, p.files...)
		if err != nil {
			return utils.Err{"cache.NewCacheFileKey", err}
		}

		f, err := p.cache.GetFile(key, i.filename)
		if err != nil {
			return utils.Err{"cache.GetFile", err}
		}
		defer f.Close()

		if !f.HasData() {
			err = f.WriteFunc(func(filename string) error {
				if err := interfaces.genInterface(name, filename); err != nil {
					return utils.Err{"genInterface", err}
				}

				// TODO: currently we need to use goimports to add missing
				// imports, we need to sort out our own imports, then we can
				// switch to gofmt.
				if err := fixup(filename); err != nil {
					return utils.Err{"fixup", err}
				}

				return nil
			})
			if err != nil {
				return utils.Err{"f.WriteFunc", err}
			}
		}

		if err := f.Install(); err != nil {
			return utils.Err{"f.Install", err}
		}
	}

	return nil
}
