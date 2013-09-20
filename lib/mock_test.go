package lib

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"path/filepath"
)

func process(filename string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("parser.ParseFile failed: %s", err))
	}

	recorders := make(map[string]string)

	return mockFile(ioutil.Discard, file, recorders)
}

func TestMockFile(t *testing.T) {
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
		if err := process(path); err != nil {
			t.Errorf("!!! FAILED !!!: %s", err)
		}

		return nil
	}

	goRoot, err := GetOutput("go", "env", "GOROOT")
	if err != nil {
		t.Fatalf("Failed to get GOROOT: %s", err)
	}

	src := filepath.Join(goRoot, "src")

	// Now use walk to process the files in src
	if err := filepath.Walk(src, fn); err != nil {
		t.Fatalf("Walk returned error: %s", err)
	}
}
