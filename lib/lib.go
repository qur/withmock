// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetOutput(name string, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	cmd := exec.Command(name, args...)
	cmd.Stderr = buf
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("External program '%s' failed (%s), with "+
			"output:\n%s", name, err, buf.String())
	}
	return strings.TrimSpace(string(out)), nil
}

func GetImports(path string, tests bool) (map[string]bool, error) {
	imports := make(map[string]bool)

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
				mock := strings.ToLower(comment) == "mock"
				if strings.HasPrefix(path, "_mock_/") {
					path = path[7:]
					mock = true
				}
				imports[path] = imports[path] || mock
			}
		}
	}

	return imports, nil
}

func GetRootImports(path string) (map[string]bool, error) {
	imports := make(map[string]bool)

	root := filepath.Join(path, "pkg/linux_amd64")

	// Add in some "magic" packages that we want to ignore
	imports["C"] = true
	imports["unsafe"] = true

	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".a") {
			return nil
		}

		imports[path[len(root)+1:len(path)-2]] = true

		return nil
	}

	err := filepath.Walk(root, fn)
	if err != nil {
		return nil, err
	}

	return imports, nil
}

func GenMockPkg(srcPath, dstRoot, name string) (map[string]bool, error) {
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
	err = MakeMock(src, dst, "")
	if err != nil {
		return nil, err
	}

	// Extract the imports from the package source
	imports, err := GetImports(src, false)
	if err != nil {
		return nil, err
	}

	// Done
	return imports, nil
}

const (
    maxNum   = 3656158440062976
	alphabet = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func base36(n int) string {
    chars := []byte(alphabet)
    res := []byte("_________a")
    for i := 9; i >= 0 && n > 0; i-- {
        res[i] = chars[n%36]
        n /= 36
    }
    return string(res)
}

func makeName(n, l int) string {
	// We want the name to always start with '_'
	if l == 1 {
		panic("We can't generate a mockName of length 1.")
	}
	l--

	base := base36(n)
	if l < len(base) {
		base = base[len(base)-l:len(base)]
	} else {
		for len(base) < l {
			base = "_" + base
		}
	}

	return "_" + base
}

var mockNames = make(map[string]bool)

func MockStandard(srcRoot, dstRoot, name string) (string, error) {
	// Figure out a new name of the same length
	count, newName := 0, makeName(0, len(name))
	for mockNames[newName] {
		count++
		newName = makeName(count, len(name))
	}

	// Write a mock version of the package
	src := filepath.Join(srcRoot, "src/pkg", name)
	dst := filepath.Join(dstRoot, "src", newName)
	err := os.MkdirAll(dst, 0700)
	if err != nil {
		return "", err
	}
	err = MakeMock(src, dst, "")
	if err != nil {
		return "", err
	}

	// Done
	return newName, nil
}

func LinkPkg(srcPath, dstRoot, name string) (map[string]bool, error) {
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
		return nil, err
	}

	// Extract the imports from the package source
	imports, err := GetImports(src, false)
	if err != nil {
		return nil, err
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

func MockImports(src, dst string, mock map[string]string) error {
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
			return mockFileImports(path, target, mock)
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
			if path == src {
				return os.MkdirAll(target, 0700)
			} else {
				return filepath.SkipDir
			}
		}

		return os.Symlink(path, target)
	}

	// Now use walk to process the files in src
	return filepath.Walk(src, fn)
}
