// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stubImports(goPath string, file *ast.File) error {
	for _, i := range file.Imports {
		imp := strings.Trim(i.Path.Value, "\"")
		name := filepath.Base(imp)
		path := filepath.Join(goPath, "src", imp)

		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}

		f, err := os.Create(filepath.Join(path, name+".go"))
		if err != nil {
			return err
		}
		defer f.Close()

		f.WriteString(fmt.Sprintf("package %s\n", name))
	}

	return nil
}

func tryLiterals(m *mockGen) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%s", p)
		}
	}()

	for name := range m.types {
		if ast.IsExported(name) {
			// We don't want to generate literals for exported types
			continue
		}
		if _, ok := m.types[name].(*ast.InterfaceType); ok {
			// We don't want to generate literals for interfaces
			continue
		}
		m.literal(name)
	}

	return nil
}

func process(filename, goPath string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("parser.ParseFile failed: %s", err))
	}

	// Create stub versions of all imports, because we are only interested in
	// the ability to parse - not if the imports actually exist.
	if err := stubImports(goPath, file); err != nil {
		return err
	}

	m := &mockGen{
		fset:      fset,
		srcPath:   filepath.Dir(filename),
		types:     make(map[string]ast.Expr),
		recorders: make(map[string]string),
		ifInfo:    newIfInfo("_ifmocks.go"),
	}
	data := &bytes.Buffer{}

	if err := m.file(data, file, filename); err != nil {
		return err
	}

	if _, err = parser.ParseFile(fset, "-", data, 0); err != nil {
		name := "_" + strings.Replace(filename, "/", "_", -1)
		f, err2 := os.Create(name)
		if err2 == nil {
			defer f.Close()
			io.Copy(f, data)
		}
		return err
	}

	return tryLiterals(m)
}

func TestMockFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "withmock-TestMockFile")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	files := []string{}

	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore directories
		if info.Mode().IsDir() {
			return nil
		}

		// Skip non-go, and test files
		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}

		files = append(files, path)

		return nil
	}

	goRoot, err := GetOutput("go", "env", "GOROOT")
	if err != nil {
		t.Fatalf("Failed to get GOROOT: %s", err)
	}

	goPath, err := GetOutput("go", "env", "GOPATH")
	if err != nil {
		t.Fatalf("Failed to get GOROOT: %s", err)
	}

	src := filepath.Join(goRoot, "src")

	// Replace GOPATH with temp directory
	os.Setenv("GOPATH", tmpDir)

	// Now use walk to process the files in src
	if err := filepath.Walk(src, fn); err != nil {
		t.Fatalf("Walk returned error: %s", err)
	}

	for i, path := range files {
		fmt.Printf("PROCESS (%d/%d): %s\n", i+1, len(files), path)
		if err := process(path, tmpDir); err != nil {
			t.Errorf("FAILED (%s):\n\t%s", path, err)
		}
	}

	os.Setenv("GOPATH", goPath)
}
