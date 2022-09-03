package codemod

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/qur/withmock/lib/extras"
)

type DstModifier struct{}

func NewDstModifier() *DstModifier {
	return &DstModifier{}
}

func (m *DstModifier) Modify(ctx context.Context, base string) ([]string, error) {
	log.Printf("MODIFY: %s", base)
	fset := token.NewFileSet()
	extraFiles := []string{}
	return extraFiles, filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		pkgs, err := decorator.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		for name, pkg := range pkgs {
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

func (m *DstModifier) processPackage(ctx context.Context, fset *token.FileSet, base string, pkg *dst.Package) ([]string, error) {
	//log.Printf("PROCESS %s: %s", base, pkg.Name)
	for path, f := range pkg.Files {
		// log.Printf("PROCESS: %s", path)
		for _, node := range f.Decls {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return nil, err
			}
			switch n := node.(type) {
			case *dst.FuncDecl:
				if !n.Name.IsExported() {
					// ignore private functions
					continue
				}
				// log.Printf("FUNC: %s (%v)", n.Name, n.Recv != nil)
				rType := ""
				var controller dst.Expr = dst.NewIdent("wmqe_main_controller")
				if n.Recv != nil {
					// TODO: set rType here to the receiver type
					if len(n.Recv.List) != 1 {
						return nil, fmt.Errorf("don't know how to handle receiver with %d fields", len(n.Recv.List))
					}
					recv := n.Recv.List[0]
					switch t := recv.Type.(type) {
					case *dst.Ident:
						if t.Path != "" {
							rType = t.Path + "." + t.Name
						} else {
							rType = t.Name
						}
					case *dst.StarExpr:
						if i, ok := t.X.(*dst.Ident); ok {
							if i.Path != "" {
								rType = i.Path + "." + i.Name
							} else {
								rType = i.Name
							}
						}
					}
					switch len(recv.Names) {
					case 0:
						recv.Names = append(recv.Names, dst.NewIdent("wmqe_self"))
						fallthrough
					case 1:
						controller = &dst.SelectorExpr{
							X:   dst.NewIdent(recv.Names[0].Name),
							Sel: dst.NewIdent("WMQE_Mock"),
						}
					default:
						return nil, fmt.Errorf("how can a receiver have multiple names? %s.%s", rType, n.Name)
					}
				}
				// log.Printf("ARGS: '%s' '%s'", rType, n.Name.Name)
				args := []dst.Expr{
					dst.NewIdent("wmqe_package"),
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + rType + `"`,
					},
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + n.Name.Name + `"`,
					},
				}
				for _, param := range n.Type.Params.List {
					for _, name := range param.Names {
						// log.Printf("ARG: %s", name.Name)
						args = append(args, dst.NewIdent(name.Name))
					}
				}
				// log.Printf("ARGS: %d, %d, %#v", len(n.Type.Params.List), len(args), args)
				var results []dst.Expr
				if n.Type.Results != nil {
					for _, result := range n.Type.Results.List {
						addResult := func() {
							results = append(results, &dst.TypeAssertExpr{
								X: &dst.IndexExpr{
									X: dst.NewIdent("ret"),
									Index: &dst.BasicLit{
										Kind:  token.INT,
										Value: strconv.FormatInt(int64(len(results)), 10),
									},
								},
								Type: dst.Clone(result.Type).(dst.Expr),
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
				if n.Body != nil {
					n.Body.List = append([]dst.Stmt{&dst.IfStmt{
						Init: &dst.AssignStmt{
							Lhs: []dst.Expr{
								dst.NewIdent("mock"),
								dst.NewIdent("ret"),
							},
							Tok: token.DEFINE,
							Rhs: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   controller,
										Sel: dst.NewIdent("MethodCalled"),
									},
									Args: args,
								},
							},
						},
						Cond: dst.NewIdent("mock"),
						Body: &dst.BlockStmt{
							List: []dst.Stmt{
								&dst.ReturnStmt{
									Results: results,
								},
							},
						},
						Decs: dst.IfStmtDecorations{
							NodeDecs: dst.NodeDecs{
								After: dst.EmptyLine,
							},
						},
					}}, n.Body.List...)
				}
			case *dst.GenDecl:
				//log.Printf("GEN: %s", n.Tok)
				if n.Tok == token.TYPE {
					for _, spec := range n.Specs {
						t := spec.(*dst.TypeSpec)
						//log.Printf("TYPE: %s (%T)", t.Name, t.Type)
						if s, ok := t.Type.(*dst.StructType); ok {
							s.Fields.List = append(s.Fields.List, &dst.Field{
								Type: dst.NewIdent("WMQE_Mock"),
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

func (*DstModifier) save(dest string, fset *token.FileSet, node *dst.File) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return decorator.Fprint(f, node)
}

func (m *DstModifier) writeExtras(base string, fset *token.FileSet, pkg string) ([]string, error) {
	path := filepath.Join(base, "wmqe_extras_"+pkg+".go")
	src, err := extras.Controller(pkg)
	if err != nil {
		return nil, err
	}
	f, err := decorator.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return []string{path}, m.save(path, fset, f)
}
