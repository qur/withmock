package codemod

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type InterfaceGenerator struct {
	prefix    string
	pkgFilter map[string]bool
}

func NewInterfaceGenerator(prefix string) *InterfaceGenerator {
	return &InterfaceGenerator{
		prefix: prefix,
	}
}

func (i *InterfaceGenerator) GenSource(ctx context.Context, mod, ver, zipfile, src, dest string) error {
	origMod, err := i.stripPrefix(mod)
	if err != nil {
		return err
	}
	mv := module.Version{Path: origMod, Version: "v" + ver}
	if err := zip.Unzip(src, mv, zipfile); err != nil {
		return fmt.Errorf("failed to unpack zip %s: %w", zipfile, err)
	}

	log.Printf("GENERATE INTERFACE: %s", src)
	fset := token.NewFileSet()
	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		if err != nil || !d.IsDir() {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return fs.SkipDir
		}
		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		for name, pkg := range pkgs {
			if err := i.processPackage(ctx, fset, origMod, path, src, dest, pkg); err != nil {
				return fmt.Errorf("failed to process %s (%s): %w", path, name, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return i.writeModFile(ctx, dest, mod)
}

func (i *InterfaceGenerator) processPackage(ctx context.Context, fset *token.FileSet, mod, path, src, dest string, pkg *ast.Package) error {
	rel, err := filepath.Rel(src, path)
	if err != nil {
		return err
	}
	pkgPath := filepath.Join(mod, rel)
	for path, f := range pkg.Files {
		if strings.HasSuffix(path, "_test.go") {
			// ignore test files
			continue
		}
		in, err := decorator.DecorateFile(fset, f)
		if err != nil {
			return err
		}
		origPkg := "wmqe_orig_" + in.Name.Name
		out := &dst.File{
			Name: dst.NewIdent(in.Name.Name),
		}
		imports := &dst.GenDecl{
			Tok: token.IMPORT,
		}
		for _, imp := range in.Imports {
			imports.Specs = append(imports.Specs, dst.Clone(imp).(*dst.ImportSpec))
		}
		out.Decls = append(out.Decls, imports, &dst.GenDecl{
			Tok: token.IMPORT,
			Specs: []dst.Spec{
				&dst.ImportSpec{
					Name: dst.NewIdent(origPkg),
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + pkgPath + `"`,
					},
				},
				&dst.ImportSpec{
					Name: dst.NewIdent("wmqe_mock"),
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"github.com/stretchr/testify/mock"`,
					},
				},
			},
			Decs: dst.GenDeclDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.EmptyLine,
					After:  dst.EmptyLine,
				},
			},
		})
		emptyLen := len(out.Decls)
		log.Printf("PROCESS: %s %s", path, pkgPath)
		for _, node := range in.Decls {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return err
			}
			switch n := node.(type) {
			case *dst.FuncDecl:
				if !n.Name.IsExported() {
					// ignore private functions and methods
					continue
				}
				if n.Recv == nil {
					t := dst.Clone(n.Type).(*dst.FuncType)
					args := []dst.Expr{}
					for _, arg := range t.Params.List {
						for _, name := range arg.Names {
							args = append(args, dst.NewIdent(name.Name))
						}
					}
					body := []dst.Stmt{}
					call := &dst.CallExpr{
						Fun: &dst.SelectorExpr{
							X:   dst.NewIdent(origPkg),
							Sel: dst.NewIdent(n.Name.Name),
						},
						Args: args,
					}
					if n.Type.Results != nil {
						body = append(body, &dst.ReturnStmt{
							Results: []dst.Expr{call},
						})
					} else {
						body = append(body, &dst.ExprStmt{
							X: call,
						})
					}
					out.Decls = append(out.Decls, &dst.FuncDecl{
						Name: dst.NewIdent(n.Name.Name),
						Type: t,
						Body: &dst.BlockStmt{
							List: body,
						},
						Decs: dst.FuncDeclDecorations{
							NodeDecs: dst.NodeDecs{
								After: dst.EmptyLine,
							},
						},
					})
				}
				t := dst.Clone(n.Type).(*dst.FuncType)
				t.Results = &dst.FieldList{
					List: []*dst.Field{
						{
							Type: &dst.StarExpr{
								X: &dst.SelectorExpr{
									X:   dst.NewIdent("wmqe_mock"),
									Sel: dst.NewIdent("Call"),
								},
							},
						},
					},
				}
				var value dst.Expr = dst.NewIdent("nil")
				typeName := ""
				if n.Recv != nil && len(n.Recv.List) > 0 {
					value = &dst.SelectorExpr{
						X:   dst.NewIdent("m"),
						Sel: dst.NewIdent("value"),
					}
					t := n.Recv.List[0].Type
					if st, ok := t.(*dst.StarExpr); ok {
						t = st.X
					} else {
						value = &dst.StarExpr{
							X: value,
						}
					}
					if i, ok := t.(*dst.Ident); ok {
						typeName = i.Name
					} else {
						return fmt.Errorf("unhandled recv expr type: %T", t)
					}
				}
				args := []dst.Expr{
					value,
					dst.NewIdent("wmqe_package"),
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + typeName + `"`,
					},
					&dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + n.Name.Name + `"`,
					},
				}
				for _, arg := range t.Params.List {
					for _, name := range arg.Names {
						args = append(args, dst.NewIdent(name.Name))
					}
				}
				body := []dst.Stmt{}
				call := &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent("wmqe_main_controller"),
						Sel: dst.NewIdent("On"),
					},
					Args: args,
				}
				anyCall := dst.Clone(call).(*dst.CallExpr)
				anyCall.Args[0] = &dst.SelectorExpr{
					X:   dst.NewIdent("wmqe_mock"),
					Sel: dst.NewIdent("Anything"),
				}
				body = append(body,
					&dst.IfStmt{
						Cond: &dst.SelectorExpr{
							X:   dst.NewIdent("m"),
							Sel: dst.NewIdent("any"),
						},
						Body: &dst.BlockStmt{
							List: []dst.Stmt{
								&dst.ReturnStmt{
									Results: []dst.Expr{
										anyCall,
									},
								},
							},
						},
					},
					&dst.ReturnStmt{
						Results: []dst.Expr{call},
					},
				)
				r := &dst.Field{
					Names: []*dst.Ident{
						dst.NewIdent("m"),
					},
				}
				if typeName == "" {
					r.Type = &dst.StarExpr{
						X: dst.NewIdent("mockPackage"),
					}
				} else {
					r.Type = &dst.StarExpr{
						X: dst.NewIdent("mock" + typeName),
					}
				}
				out.Decls = append(out.Decls, &dst.FuncDecl{
					Recv: &dst.FieldList{
						List: []*dst.Field{r},
					},
					Name: dst.NewIdent(n.Name.Name),
					Type: t,
					Body: &dst.BlockStmt{
						List: body,
					},
					Decs: dst.FuncDeclDecorations{
						NodeDecs: dst.NodeDecs{
							After: dst.EmptyLine,
						},
					},
				})
			case *dst.GenDecl:
				log.Printf("GEN: %s", n.Tok)
				s := []dst.Spec{}
				switch n.Tok {
				case token.TYPE:
					for _, spec := range n.Specs {
						t := spec.(*dst.TypeSpec)
						out.Decls = append(out.Decls,
							&dst.FuncDecl{
								Recv: &dst.FieldList{
									List: []*dst.Field{
										{
											Type: &dst.StarExpr{
												X: dst.NewIdent("wmqe_mock"),
											},
										},
									},
								},
								Name: dst.NewIdent("This" + t.Name.Name),
								Type: &dst.FuncType{
									Params: &dst.FieldList{
										List: []*dst.Field{
											{
												Names: []*dst.Ident{
													dst.NewIdent("value"),
												},
												Type: &dst.StarExpr{
													X: dst.NewIdent(t.Name.Name),
												},
											},
										},
									},
									Results: &dst.FieldList{
										List: []*dst.Field{
											{
												Type: &dst.StarExpr{
													X: dst.NewIdent("mock" + t.Name.Name),
												},
											},
										},
									},
								},
								Body: &dst.BlockStmt{
									List: []dst.Stmt{
										&dst.ReturnStmt{
											Results: []dst.Expr{
												&dst.UnaryExpr{
													Op: token.AND,
													X: &dst.CompositeLit{
														Type: dst.NewIdent("mock" + t.Name.Name),
														Elts: []dst.Expr{
															&dst.KeyValueExpr{
																Key:   dst.NewIdent("any"),
																Value: dst.NewIdent("false"),
															},
															&dst.KeyValueExpr{
																Key:   dst.NewIdent("value"),
																Value: dst.NewIdent("value"),
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&dst.FuncDecl{
								Recv: &dst.FieldList{
									List: []*dst.Field{
										{
											Type: &dst.StarExpr{
												X: dst.NewIdent("wmqe_mock"),
											},
										},
									},
								},
								Name: dst.NewIdent("Any" + t.Name.Name),
								Type: &dst.FuncType{
									Params: &dst.FieldList{},
									Results: &dst.FieldList{
										List: []*dst.Field{
											{
												Type: &dst.StarExpr{
													X: dst.NewIdent("mock" + t.Name.Name),
												},
											},
										},
									},
								},
								Body: &dst.BlockStmt{
									List: []dst.Stmt{
										&dst.ReturnStmt{
											Results: []dst.Expr{
												&dst.UnaryExpr{
													Op: token.AND,
													X: &dst.CompositeLit{
														Type: dst.NewIdent("mock" + t.Name.Name),
														Elts: []dst.Expr{
															&dst.KeyValueExpr{
																Key:   dst.NewIdent("any"),
																Value: dst.NewIdent("true"),
															},
															&dst.KeyValueExpr{
																Key:   dst.NewIdent("value"),
																Value: dst.NewIdent("nil"),
															},
														},
													},
												},
											},
										},
									},
								},
							},
							&dst.FuncDecl{
								Recv: &dst.FieldList{
									List: []*dst.Field{
										{
											Names: []*dst.Ident{
												dst.NewIdent("m"),
											},
											Type: &dst.StarExpr{
												X: dst.NewIdent("mock" + t.Name.Name),
											},
										},
									},
								},
								Name: dst.NewIdent("On"),
								Type: &dst.FuncType{
									Params: &dst.FieldList{
										List: []*dst.Field{
											{
												Names: []*dst.Ident{
													dst.NewIdent("method"),
												},
												Type: dst.NewIdent("string"),
											},
											{
												Names: []*dst.Ident{
													dst.NewIdent("args"),
												},
												Type: &dst.Ellipsis{
													Elt: dst.NewIdent("any"),
												},
											},
										},
									},
									Results: &dst.FieldList{
										List: []*dst.Field{
											{
												Type: &dst.StarExpr{
													X: &dst.SelectorExpr{
														X:   dst.NewIdent("wmqe_mock"),
														Sel: dst.NewIdent("Call"),
													},
												},
											},
										},
									},
								},
								Body: &dst.BlockStmt{
									List: []dst.Stmt{
										&dst.IfStmt{
											Cond: &dst.SelectorExpr{
												X:   dst.NewIdent("m"),
												Sel: dst.NewIdent("any"),
											},
											Body: &dst.BlockStmt{
												List: []dst.Stmt{
													&dst.ReturnStmt{
														Results: []dst.Expr{
															&dst.CallExpr{
																Fun: &dst.SelectorExpr{
																	X:   dst.NewIdent("wmqe_main_controller"),
																	Sel: dst.NewIdent("On"),
																},
																Args: []dst.Expr{
																	&dst.SelectorExpr{
																		X:   dst.NewIdent("wmqe_mock"),
																		Sel: dst.NewIdent("Anything"),
																	},
																	dst.NewIdent("wmqe_package"),
																	&dst.BasicLit{
																		Kind:  token.STRING,
																		Value: `"` + t.Name.Name + `"`,
																	},
																	dst.NewIdent("method"),
																	dst.NewIdent("args"),
																},
																Ellipsis: true,
															},
														},
													},
												},
											},
										},
										&dst.ReturnStmt{
											Results: []dst.Expr{
												&dst.CallExpr{
													Fun: &dst.SelectorExpr{
														X:   dst.NewIdent("wmqe_main_controller"),
														Sel: dst.NewIdent("On"),
													},
													Args: []dst.Expr{
														&dst.SelectorExpr{
															X:   dst.NewIdent("m"),
															Sel: dst.NewIdent("value"),
														},
														dst.NewIdent("wmqe_package"),
														&dst.BasicLit{
															Kind:  token.STRING,
															Value: `"` + t.Name.Name + `"`,
														},
														dst.NewIdent("method"),
														dst.NewIdent("args"),
													},
													Ellipsis: true,
												},
											},
										},
									},
								},
							},
						)
						s = append(s, &dst.TypeSpec{
							Name: dst.NewIdent("mock" + t.Name.Name),
							Type: &dst.StructType{
								Fields: &dst.FieldList{
									List: []*dst.Field{
										{
											Names: []*dst.Ident{
												dst.NewIdent("any"),
											},
											Type: dst.NewIdent("bool"),
										},
										{
											Names: []*dst.Ident{
												dst.NewIdent("value"),
											},
											Type: &dst.StarExpr{
												X: &dst.SelectorExpr{
													X:   dst.NewIdent(origPkg),
													Sel: dst.NewIdent(t.Name.Name),
												},
											},
										},
									},
								},
							},
						})
						if !t.Name.IsExported() {
							continue
						}
						log.Printf("TYPE: %s", t.Name.Name)
						s = append(s, &dst.TypeSpec{
							Name:   dst.NewIdent(t.Name.Name),
							Assign: true,
							Type: &dst.SelectorExpr{
								X:   dst.NewIdent(origPkg),
								Sel: dst.NewIdent(t.Name.Name),
							},
						})
					}
				case token.VAR, token.CONST:
					for _, spec := range n.Specs {
						v := spec.(*dst.ValueSpec)
						names := make([]string, 0, len(v.Names))
						for _, n := range v.Names {
							if n.IsExported() {
								log.Printf("VAR: %s", n.Name)
								names = append(names, n.Name)
							}
						}
						if len(names) == 0 {
							continue
						}
						nv := &dst.ValueSpec{}
						for _, name := range names {
							nv.Names = append(nv.Names, dst.NewIdent(name))
							nv.Values = append(nv.Values, &dst.SelectorExpr{
								X:   dst.NewIdent(origPkg),
								Sel: dst.NewIdent(name),
							})
						}
						s = append(s, nv)
					}
				}
				if len(s) > 0 {
					out.Decls = append(out.Decls, &dst.GenDecl{
						Tok:   n.Tok,
						Specs: s,
						Decs: dst.GenDeclDecorations{
							NodeDecs: dst.NodeDecs{
								After: dst.EmptyLine,
							},
						},
					})
				}
			}
		}
		if len(out.Decls) == emptyLen {
			// nothing public was added, so skip the whole file
			continue
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to resolve extra path %s: %s", path, err)
		}
		if err := i.save(filepath.Join(dest, rel), fset, out); err != nil {
			return fmt.Errorf("failed to format %s: %w", path, err)
		}
	}
	return i.writeExtras(ctx, fset, mod, path, src, dest, pkg)
}

func (i *InterfaceGenerator) stripPrefix(mod string) (string, error) {
	if !strings.HasPrefix(mod, i.prefix) {
		return "", fmt.Errorf("module '%s' didn't have prefix '%s'", mod, i.prefix)
	}
	return mod[len(i.prefix):], nil
}

func (*InterfaceGenerator) save(dest string, fset *token.FileSet, node *dst.File) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return decorator.Fprint(f, node)
}

func (i *InterfaceGenerator) writeExtras(ctx context.Context, fset *token.FileSet, mod, path, src, dest string, pkg *ast.Package) error {
	rel, err := filepath.Rel(src, path)
	if err != nil {
		return err
	}
	pkgPath := filepath.Join(mod, rel)
	origPkg := "wmqe_orig_" + pkg.Name
	out := &dst.File{
		Name: dst.NewIdent(pkg.Name),
	}
	out.Decls = append(out.Decls,
		&dst.GenDecl{
			Tok: token.IMPORT,
			Specs: []dst.Spec{
				&dst.ImportSpec{
					Name: dst.NewIdent(origPkg),
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + pkgPath + `"`,
					},
				},
				&dst.ImportSpec{
					Name: dst.NewIdent("wmqe_ctrl"),
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"github.com/stretchr/testify/mock"`,
						// Value: `"gowm.in/ctrl"`,
					},
				},
			},
			Decs: dst.GenDeclDecorations{
				NodeDecs: dst.NodeDecs{
					Before: dst.EmptyLine,
					After:  dst.EmptyLine,
				},
			},
		},
		&dst.GenDecl{
			Tok: token.VAR,
			Specs: []dst.Spec{
				&dst.ValueSpec{
					Names: []*dst.Ident{
						dst.NewIdent("wmqe_main_controller"),
					},
					Values: []dst.Expr{
						&dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   dst.NewIdent("wmqe_ctrl"),
								Sel: dst.NewIdent("DefaultController"),
							},
						},
					},
				},
			},
		},
		&dst.FuncDecl{
			Name: dst.NewIdent("init"),
			Type: &dst.FuncType{
				Params: &dst.FieldList{},
			},
			Body: &dst.BlockStmt{
				List: []dst.Stmt{
					&dst.ExprStmt{
						X: &dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   dst.NewIdent(origPkg),
								Sel: dst.NewIdent("WMQE_SetController"),
							},
							Args: []dst.Expr{
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   dst.NewIdent("wmqe_ctrl"),
										Sel: dst.NewIdent("DefaultController"),
									},
								},
							},
						},
					},
				},
			},
		},
	)
	if err := i.save(filepath.Join(dest, "wmqe_extras_"+pkg.Name+".go"), fset, out); err != nil {
		return fmt.Errorf("failed to format %s: %w", path, err)
	}
	return nil
}

func (*InterfaceGenerator) writeModFile(ctx context.Context, dest, mod string) error {
	log.Printf("MODFILE: %s", dest)

	mf := &modfile.File{}
	if err := mf.AddModuleStmt(mod); err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	data, err := mf.Format()
	if err != nil {
		return fmt.Errorf("failed to format go.mod for %s: %w", dest, err)
	}

	f, err := os.Create(filepath.Join(dest, "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write go.mod for %s: %w", dest, err)
	}

	buf := &bytes.Buffer{}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dest
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy modfile for %s: %w:\n%s", dest, err, buf.String())
	}
	return nil
}
