package lib

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"path/filepath"
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

	recorders := make(map[string]string)

	dir := filepath.Dir(filename)
	return mockFile(ioutil.Discard, dir, file, recorders)
}

func TestMockFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "withmock-TestMockFile")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}
	os.RemoveAll(tmpDir)

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

		t.Logf("PROCESS: %s", path)
		if err := process(path, tmpDir); err != nil {
			t.Errorf("!!! FAILED !!!: %s", err)
		}

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

	os.Setenv("GOPATH", goPath)
}
