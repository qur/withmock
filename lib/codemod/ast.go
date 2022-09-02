package codemod

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/qur/withmock/lib/extras"
)

type AstModifier struct{}

func NewAstModifier() *AstModifier {
	return &AstModifier{}
}

func (m *AstModifier) Modify(ctx context.Context, base string) ([]string, error) {
	log.Printf("MODIFY: %s", base)
	fset := token.NewFileSet()
	extraFiles := []string{}
	return extraFiles, filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkgs, err := parser.ParseDir(fset, path, nil, 0 /* parser.ParseComments */)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		for name, pkg := range pkgs {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return err
			}
			extras, err := m.processPackage(ctx, fset, path, pkg)
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

func (m *AstModifier) processPackage(ctx context.Context, fset *token.FileSet, base string, pkg *ast.Package) ([]string, error) {
	//log.Printf("PROCESS %s: %s", base, pkg.Name)
	for path, f := range pkg.Files {
		log.Printf("PROCESS: %s", path)
		for _, node := range f.Decls {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return nil, err
			}
			switch n := node.(type) {
			case *ast.FuncDecl:
				if !n.Name.IsExported() {
					// ignore private functions
					continue
				}
				// log.Printf("FUNC: %s (%v)", n.Name, n.Recv != nil)
				rType := ""
				if n.Recv != nil {
					// TODO: set rType here to the receiver type
				}
				// log.Printf("ARGS: '%s' '%s'", rType, n.Name.Name)
				args := []ast.Expr{
					ast.NewIdent("wmqe_package"),
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: `"` + rType + `"`,
					},
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: `"` + n.Name.Name + `"`,
					},
				}
				for _, param := range n.Type.Params.List {
					for _, name := range param.Names {
						// log.Printf("ARG: %s", name.Name)
						args = append(args, ast.NewIdent(name.Name))
					}
				}
				// log.Printf("ARGS: %d, %d, %#v", len(n.Type.Params.List), len(args), args)
				var results []ast.Expr
				if n.Type.Results != nil {
					for _, result := range n.Type.Results.List {
						addResult := func() {
							results = append(results, &ast.TypeAssertExpr{
								X: &ast.IndexExpr{
									X: ast.NewIdent("ret"),
									Index: &ast.BasicLit{
										Kind:  token.INT,
										Value: strconv.FormatInt(int64(len(results)), 10),
									},
								},
								Type: result.Type,
							})
						}
						if len(result.Names) == 0 {
							addResult()
						} else {
							for range result.Names {
								addResult()
							}
						}
					}
				}
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
								Args: args,
							},
						},
					},
					Cond: ast.NewIdent("mock"),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: results,
							},
						},
					},
				}}, n.Body.List...)
			case *ast.GenDecl:
				//log.Printf("GEN: %s", n.Tok)
				if n.Tok == token.TYPE {
					for _, spec := range n.Specs {
						t := spec.(*ast.TypeSpec)
						if !t.Name.IsExported() {
							// ignore private types
							continue
						}
						//log.Printf("TYPE: %s (%T)", t.Name, t.Type)
						if s, ok := t.Type.(*ast.StructType); ok {
							s.Fields.List = append(s.Fields.List, &ast.Field{
								Type: ast.NewIdent("WMQE_Mock"),
							})
						}
					}
				}
			}
		}
		if err := m.save(path, fset, f); err != nil {
			return nil, fmt.Errorf("failed to format %s: %w", path, err)
		}
	}
	//log.Printf("EXTRAS FOR %s", pkg.Name)
	return m.writeExtras(base, fset, pkg.Name)
}

func (*AstModifier) save(dest string, fset *token.FileSet, node any) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return format.Node(f, fset, node)
}

func (m *AstModifier) writeExtras(base string, fset *token.FileSet, pkg string) ([]string, error) {
	path := filepath.Join(base, "wmqe_extras_"+pkg+".go")
	src, err := extras.Controller(pkg)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return []string{path}, m.save(path, fset, f)
}
