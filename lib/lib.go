// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func LookupImportPath(impPath string) (string, error) {
	if strings.HasPrefix(impPath, "_/") {
		// special case if impPath is outside of GOPATH
		return impPath[1:], nil
	}

	path, err := GetOutput("go", "list", "-e", "-f", "{{.Dir}}", impPath)
	if err != nil {
		return "", err
	}

	if path == "" {
		return "", fmt.Errorf("Unable to find package: %s", impPath)
	}

	return path, nil
}

func GetOutput(name string, args ...string) (string, error) {
	return GetCmdOutput(exec.Command(name, args...))
}

func GetCmdOutput(cmd *exec.Cmd) (string, error) {
	buf := &bytes.Buffer{}
	cmd.Stderr = buf
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("External program '%s' failed (%s), with "+
			"output:\n%s", cmd.Args[0], err, buf.String())
	}
	return strings.TrimSpace(string(out)), nil
}

func hasNonGoCode(impPath string) (bool, error) {
	src, err := LookupImportPath(impPath)
	if err != nil {
		return false, err
	}

	nonGoCode := false

	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore every directory except path
		if info.Mode().IsDir() {
			if path == src {
				return nil
			} else {
				return filepath.SkipDir
			}
		}

		// Non-code we leave alone, code may need modification
		if strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".s") {
			nonGoCode = true
		}

		return nil
	}

	// Now use walk to process the files in src
	return nonGoCode, filepath.Walk(src, fn)
}

func GetImports(path string, tests bool) (importSet, error) {
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
	pkgs, err := parser.ParseDir(fset, path, isGoFile,
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
					path2 = comment[8 : len(comment)-1]
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

func GetMockedPackages(path string) (map[string]string, error) {
	imports := make(map[string]string)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil,
		parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, i := range file.Imports {
		impPath := strings.Trim(i.Path.Value, "\"")
		comment := strings.TrimSpace(i.Comment.Text())
		mock := strings.ToLower(comment) == "mock"
		if strings.HasPrefix(impPath, "_mock_/") {
			mock = true
		}

		if !mock {
			continue
		}

		if i.Name != nil {
			imports[i.Name.String()] = impPath
		} else {
			// TODO: pkgName for vendor paths?
			name, err := getPackageName(impPath, filepath.Dir(path), "")
			if err != nil {
				return nil, err
			}
			imports[name] = impPath
		}
	}

	return imports, nil
}

func getStdlibImports(path string) (map[string]bool, error) {
	imports := make(map[string]bool)

	list, err := GetOutput("go", "list", "std")
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(list, "\n") {
		imports[strings.TrimSpace(line)] = true
	}

	// Add in some "magic" packages that we want to ignore
	imports["C"] = true

	return imports, nil
}

// Import "marks":
//
//  _ : mock
//  + : normal (no mark actually applied)
//  @ : test
//  = : replace
type mark string

const (
	noMark      mark = ""
	normalMark  mark = "+"
	mockMark    mark = "_"
	testMark    mark = "@"
	replaceMark mark = "="
)

func markImport(name string, m mark) string {
	switch m {
	case noMark, normalMark:
		return name
	case mockMark, testMark, replaceMark:
		return string(m) + name[1:]
	default:
		panic(fmt.Sprintf("Unknown import mark: %s", m))
	}
}

func getMark(label string) mark {
	switch label[0] {
	case mockMark[0]:
		return mockMark
	case testMark[0]:
		return testMark
	case replaceMark[0]:
		return replaceMark
	default:
		return normalMark
	}
}

func GenPkg(srcPath, dstRoot, name string, mock bool, cfg *MockConfig) (importSet, error) {
	log.Printf("GenPkg: srcPath:%s, dstRoot:%s, name:%s, mock:%v", srcPath, dstRoot, name, mock)
	// Find the package source, it may be in any entry in srcPath
	srcRoot := ""
	for _, src := range filepath.SplitList(srcPath) {
		if exists(filepath.Join(src, "src", name)) {
			srcRoot = src
			break
		}
	}
	if srcRoot == "" {
		return nil, fmt.Errorf("Package '%s' not found in any of '%s'.", name,
			srcPath)
	}

	// Write a mock version of the package
	src := filepath.Join(srcRoot, "src", name)
	dst := filepath.Join(dstRoot, "src", name)
	err := os.MkdirAll(dst, 0700)
	if err != nil {
		return nil, err
	}
	cfg.MatchOSArch = true
	imports, err := MakePkg(src, dst, name, mock, cfg)
	if err != nil {
		return nil, err
	}

	// Done
	return imports, nil
}

func MockStandard(srcRoot, dstRoot, name string, cfg *MockConfig) error {
	log.Printf("MockStandard: src: %s, dst: %s, name: %s", srcRoot, dstRoot, name)
	// Write a mock version of the package
	var src string
	if _, err := os.Stat(srcRoot + "/src/pkg"); err == nil {
		src = filepath.Join(srcRoot, "src/pkg", name)
	} else {
		src = filepath.Join(srcRoot, "src", name)
	}
	dst := filepath.Join(dstRoot, "src", markImport(name, mockMark))
	err := os.MkdirAll(dst, 0700)
	if err != nil {
		return Cerr{"MkdirAll", err}
	}
	cfg.MockPrototypes = true
	cfg.IgnoreInits = true
	cfg.MatchOSArch = true
	cfg.IgnoreNonGoFiles = true

	// Some versions of go include a vendor folder ...
	// TODO: check it exists first
	vsrc := filepath.Join(srcRoot, "src", "vendor")
	vdst := filepath.Join(dst, "vendor")
	if _, err := os.Stat(vsrc); err == nil {
		// stdlib has a vendor directory, so symlink it
		log.Printf("vendor: src: %s, dst: %s", vsrc, vdst)
		lnk, err := os.Readlink(vdst)
		if os.IsNotExist(err) {
			if err := os.Symlink(vsrc, vdst); err != nil {
				return Cerr{"Vendor Symlink", err}
			}
		} else if err != nil {
			return Cerr{"Check vendor link", err}
		} else if lnk != vsrc {
			err := fmt.Errorf("Vendor link wrong: got %s, want %s", lnk, vsrc)
			return Cerr{"Check vendor link", err}
		}
	}

	_, err = MakePkg(src, dst, name, true, cfg)
	if err != nil {
		return Cerr{"MakePkg", err}
	}

	// Done
	return nil
}

func ReplacePkg(srcPath, dstRoot, from, as string) (importSet, error) {
	// Find the package source, it may be in any entry in srcPath
	srcRoot := ""
	for _, src := range filepath.SplitList(srcPath) {
		if exists(filepath.Join(src, "src", from)) {
			srcRoot = src
			break
		}
	}
	if srcRoot == "" {
		return nil, fmt.Errorf("Package '%s' not found in any of '%s'.", from,
			srcPath)
	}

	// Copy the package source
	src := filepath.Join(srcRoot, "src", from)
	dst := filepath.Join(dstRoot, "src", as)
	err := symlinkPackage(src, dst)
	if err != nil {
		return nil, Cerr{"symlinkPackage", err}
	}

	// Extract the imports from the package source
	imports, err := GetImports(src, false)
	if err != nil {
		return nil, Cerr{"GetImports", err}
	}

	// Done
	return imports, nil
}

func LinkPkg(srcPath, dstRoot, name string) (importSet, error) {
	// Find the package source, it may be in any entry in srcPath
	srcRoot := ""
	for _, src := range filepath.SplitList(srcPath) {
		if exists(filepath.Join(src, "src", name)) {
			srcRoot = src
			break
		}
	}
	if srcRoot == "" {
		return nil, fmt.Errorf("Package '%s' not found in any of '%s'.", name,
			srcPath)
	}

	// Copy the package source
	src := filepath.Join(srcRoot, "src", name)
	dst := filepath.Join(dstRoot, "src", name)
	err := symlinkPackage(src, dst)
	if err != nil {
		return nil, Cerr{"symlinkPackage", err}
	}

	// Extract the imports from the package source
	imports, err := GetImports(src, false)
	if err != nil {
		return nil, Cerr{"GetImports", err}
	}

	// Done
	return imports, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic(err)
}

func MockImports(src, dst string, names map[string]string, cfg *Config) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		// Ignore every directory except src (which we need to mirror)
		if info.Mode().IsDir() {
			if path == src {
				return os.MkdirAll(target, 0700)
			} else {
				return filepath.SkipDir
			}
		}

		// Non-code we leave alone, code may need modification
		if !strings.HasSuffix(path, ".go") {
			return os.Symlink(path, target)
		} else {
			return mockFileImports(path, target, names, cfg)
		}
	}

	// Now use walk to process the files in src
	return filepath.Walk(src, fn)
}

func symlinkPackage(src, dst string) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		// Ignore every directory except src (which we need to mirror)
		if info.Mode().IsDir() {
			return os.MkdirAll(target, 0700)
		}

		return os.Symlink(path, target)
	}

	// Now use walk to process the files in src
	return filepath.Walk(src, fn)
}
