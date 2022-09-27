package codemod

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/qur/withmock/lib/extras"
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

func (i *InterfaceGenerator) GenMod(ctx context.Context, mod, ver, src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	mf, err := modfile.Parse(src, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", src, err)
	}
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dest, err)
	}
	defer f.Close()
	if err := extras.InterfaceModFile(mod, ver, mf.Go.Version, f); err != nil {
		f.Close()
		return fmt.Errorf("failed to write %s: %w", dest, err)
	}
	return nil
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
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
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
	})
}

func (i *InterfaceGenerator) processPackage(ctx context.Context, fset *token.FileSet, mod, path, src, dest string, pkg *ast.Package) error {
	rel, err := filepath.Rel(src, path)
	if err != nil {
		return err
	}
	pkgPath := filepath.Join(mod, rel)
	for path, f := range pkg.Files {
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
				if !n.Name.IsExported() || n.Recv != nil {
					// ignore private functions or methods
					continue
				}
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
			case *dst.GenDecl:
				log.Printf("GEN: %s", n.Tok)
				s := []dst.Spec{}
				switch n.Tok {
				case token.TYPE:
					for _, spec := range n.Specs {
						t := spec.(*dst.TypeSpec)
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
	return nil
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
