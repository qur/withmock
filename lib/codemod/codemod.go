package codemod

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/qur/withmock/lib/extras"
)

type Modifier struct{}

func NewModifier() *Modifier {
	return &Modifier{}
}

func (m *Modifier) Modify(base string) ([]string, error) {
	log.Printf("MODIFY: %s", base)
	fset := token.NewFileSet()
	extraFiles := []string{}
	return extraFiles, filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		for name, pkg := range pkgs {
			extras, err := processPackage(fset, path, pkg)
			if err != nil {
				return fmt.Errorf("failed to process %s (%s): %w", path, name, err)
			}
			for _, path := range extras {
				extra, err := filepath.Rel(base, path)
				if err != nil {
					return fmt.Errorf("failed to resolve extra path %s: %s", path, err)
				}
				extraFiles = append(extraFiles, extra)
			}
		}
		return nil
	})
}

func processPackage(fset *token.FileSet, base string, pkg *ast.Package) ([]string, error) {
	log.Printf("PROCESS %s: %s", base, pkg.Name)
	for path, f := range pkg.Files {
		log.Printf("PROCESS: %s", path)
		if err := save(path, fset, f); err != nil {
			return nil, fmt.Errorf("failed to format %s: %w", path, err)
		}
	}
	return writeExtras(base, fset, pkg.Name)
}

func save(dest string, fset *token.FileSet, node any) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return format.Node(f, fset, node)
}

func writeExtras(base string, fset *token.FileSet, pkg string) ([]string, error) {
	path := filepath.Join(base, "wmqe_extras.go")
	src, err := extras.Controller(pkg)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return []string{path}, save(path, fset, f)
}
