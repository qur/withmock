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
)

type Modifier struct{}

func NewModifier() *Modifier {
	return &Modifier{}
}

func (m *Modifier) Modify(path string) error {
	log.Printf("MODIFY: %s", path)
	fset := token.NewFileSet()
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		for name, pkg := range pkgs {
			if err := processPackage(fset, path, pkg); err != nil {
				return fmt.Errorf("failed to process %s (%s): %w", path, name, err)
			}
		}
		return nil
	})
}

func processPackage(fset *token.FileSet, base string, pkg *ast.Package) error {
	log.Printf("PROCESS %s: %s", base, pkg.Name)
	for path, f := range pkg.Files {
		log.Printf("PROCESS: %s", path)
		if err := save(path, fset, f); err != nil {
			return fmt.Errorf("failed to format %s: %w", path, err)
		}
	}
	return nil
}

func save(dest string, fset *token.FileSet, node any) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return format.Node(f, fset, node)
}
