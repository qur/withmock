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
		for _, node := range f.Decls {
			switch n := node.(type) {
			case *ast.FuncDecl:
				if !n.Name.IsExported() {
					// ignore private functions
					continue
				}
				log.Printf("FUNC: %s (%v)", n.Name, n.Recv != nil)
				n.Body.List = append([]ast.Stmt{&ast.IfStmt{
					Init: &ast.AssignStmt{
						Lhs: []ast.Expr{
							ast.NewIdent("mock"),
							ast.NewIdent("ret"),
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent("wmqe_main_controller"),
									Sel: ast.NewIdent("MethodCalled"),
								},
								Args: []ast.Expr{
									ast.NewIdent("wmqe_package"),
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: `""`,
									},
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: `"` + n.Name.Name + `"`,
									},
								},
							},
						},
					},
					Cond: ast.NewIdent("mock"),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: []ast.Expr{&ast.TypeAssertExpr{
									X: &ast.IndexExpr{
										X: ast.NewIdent("ret"),
										Index: &ast.BasicLit{
											Kind:  token.INT,
											Value: "0",
										},
									},
									Type: ast.NewIdent("string"),
								}},
							},
						},
					},
				}}, n.Body.List...)
			case *ast.GenDecl:
				log.Printf("GEN: %s", n.Tok)
				if n.Tok == token.TYPE {
					for _, spec := range n.Specs {
						t := spec.(*ast.TypeSpec)
						if !t.Name.IsExported() {
							// ignore private types
							continue
						}
						log.Printf("TYPE: %s (%T)", t.Name, t.Type)
						if s, ok := t.Type.(*ast.StructType); ok {
							s.Fields.List = append(s.Fields.List, &ast.Field{
								Type: ast.NewIdent("WMQE_Mock"),
							})
						}
					}
				}
			}
		}
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
