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
	"github.com/qur/withmock/lib/proxy/api"
	"github.com/qur/withmock/lib/proxy/modify"
)

type rawMock struct {
	name    string
	methods []*dst.Field
}

type rawMockFile struct {
	path    string
	pkgName string
	imports []*dst.ImportSpec
	mocks   []rawMock
}

type MockGenerator struct {
	prefix    string
	pkgFilter map[string]bool
}

func NewMockGenerator(prefix string) *MockGenerator {
	return &MockGenerator{
		prefix: prefix,
	}
}

func (i *MockGenerator) GenModMode() modify.GenModMode {
	return modify.GenModFromModfile
}

func (i *MockGenerator) GenMod(ctx context.Context, mod, ver, src, dest string) error {
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

func (i *MockGenerator) GenSource(ctx context.Context, mod, ver, zipfile, src, dest string) error {
	allMocks := []rawMockFile{}
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
			mocks, err := i.processPackage(ctx, fset, origMod, path, src, dest, pkg)
			if err != nil {
				return fmt.Errorf("failed to process %s (%s): %w", path, name, err)
			}
			allMocks = append(allMocks, mocks...)
		}
		return nil
	}); err != nil {
		return err
	}
	if len(allMocks) == 0 {
		return api.UnknownVersion(mod, ver)
	}
	return i.renderMocks(ctx, fset, allMocks)
}

func (i *MockGenerator) processPackage(ctx context.Context, fset *token.FileSet, mod, path, src, dest string, pkg *ast.Package) ([]rawMockFile, error) {
	files := []rawMockFile{}
	rel, err := filepath.Rel(src, path)
	if err != nil {
		return files, err
	}
	pkgPath := filepath.Join(mod, rel)
	for path, f := range pkg.Files {
		in, err := decorator.DecorateFile(fset, f)
		if err != nil {
			return files, err
		}
		mocks := []rawMock{}
		log.Printf("PROCESS: %s %s", path, pkgPath)
		for _, node := range in.Decls {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return files, err
			}
			n, ok := node.(*dst.GenDecl)
			if !ok || n.Tok != token.TYPE {
				// not a type decl
				continue
			}
			for _, spec := range n.Specs {
				t := spec.(*dst.TypeSpec)
				ift, ok := t.Type.(*dst.InterfaceType)
				if !t.Name.IsExported() || !ok || ift.Methods == nil || len(ift.Methods.List) == 0 {
					continue
				}
				methods := []*dst.Field{}
				log.Printf("TYPE: %s %#v", t.Name.Name, ift)
				for _, f := range ift.Methods.List {
					log.Printf("METHOD: %s %T", f.Names, f.Type)
					methods = append(methods, dst.Clone(f).(*dst.Field))
				}
				mocks = append(mocks, rawMock{
					name:    "Mock" + t.Name.Name,
					methods: methods,
				})
			}
		}
		if len(mocks) == 0 {
			// no mocks were added, so skip the whole file
			continue
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return files, fmt.Errorf("failed to resolve extra path %s: %s", path, err)
		}
		files = append(files, rawMockFile{
			path:    filepath.Join(dest, rel),
			pkgName: in.Name.Name,
			imports: in.Imports,
			mocks:   mocks,
		})
	}
	return files, nil
}

func (i *MockGenerator) processPackageOld(ctx context.Context, fset *token.FileSet, mod, path, src, dest string, pkg *ast.Package) (int, error) {
	output := 0
	rel, err := filepath.Rel(src, path)
	if err != nil {
		return output, err
	}
	pkgPath := filepath.Join(mod, rel)
	for path, f := range pkg.Files {
		in, err := decorator.DecorateFile(fset, f)
		if err != nil {
			return output, err
		}
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
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"gowm.in/ctrl/mock"`,
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
				return output, err
			}
			n, ok := node.(*dst.GenDecl)
			if !ok || n.Tok != token.TYPE {
				// not a type decl
				continue
			}
			for _, spec := range n.Specs {
				t := spec.(*dst.TypeSpec)
				ift, ok := t.Type.(*dst.InterfaceType)
				if !t.Name.IsExported() || !ok {
					continue
				}
				methods := []dst.Decl{}
				log.Printf("TYPE: %s %#v", t.Name.Name, ift)
				for _, f := range ift.Methods.List {
					log.Printf("METHOD: %s %T", f.Names, f.Type)
					if ft, ok := f.Type.(*dst.FuncType); ok {
						methods = append(methods, &dst.FuncDecl{
							Recv: &dst.FieldList{
								List: []*dst.Field{
									{
										Names: []*dst.Ident{
											dst.NewIdent("m"),
										},
										Type: &dst.StarExpr{
											X: dst.NewIdent("Mock" + t.Name.Name),
										},
									},
								},
							},
							Name: dst.NewIdent(f.Names[0].Name),
							Type: dst.Clone(ft).(*dst.FuncType),
						})
					}
				}
				out.Decls = append(out.Decls, &dst.GenDecl{
					Tok: n.Tok,
					Specs: []dst.Spec{
						&dst.TypeSpec{
							Name: dst.NewIdent("Mock" + t.Name.Name),
							Type: &dst.StructType{
								Fields: &dst.FieldList{
									List: []*dst.Field{
										{
											Type: &dst.SelectorExpr{
												X:   dst.NewIdent("mock"),
												Sel: dst.NewIdent("Mock"),
											},
										},
									},
								},
							},
						},
					},
					Decs: dst.GenDeclDecorations{
						NodeDecs: dst.NodeDecs{
							After: dst.EmptyLine,
						},
					},
				})
				if len(methods) > 0 {
					out.Decls = append(out.Decls, methods...)
				}
			}
		}
		if len(out.Decls) == emptyLen {
			// nothing public was added, so skip the whole file
			continue
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return output, fmt.Errorf("failed to resolve extra path %s: %s", path, err)
		}
		if err := i.save(filepath.Join(dest, rel), fset, out); err != nil {
			return output, fmt.Errorf("failed to format %s: %w", path, err)
		}
		output++
	}
	return output, nil
}

func (i *MockGenerator) stripPrefix(mod string) (string, error) {
	if !strings.HasPrefix(mod, i.prefix) {
		return "", fmt.Errorf("module '%s' didn't have prefix '%s'", mod, i.prefix)
	}
	return mod[len(i.prefix):], nil
}

func (*MockGenerator) save(dest string, fset *token.FileSet, node *dst.File) error {
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

func (i *MockGenerator) renderMocks(ctx context.Context, fset *token.FileSet, files []rawMockFile) error {
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		out := &dst.File{
			Name: dst.NewIdent(file.pkgName),
		}
		imports := &dst.GenDecl{
			Tok: token.IMPORT,
		}
		for _, imp := range file.imports {
			imports.Specs = append(imports.Specs, dst.Clone(imp).(*dst.ImportSpec))
		}
		out.Decls = append(out.Decls, imports, &dst.GenDecl{
			Tok: token.IMPORT,
			Specs: []dst.Spec{
				&dst.ImportSpec{
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"gowm.in/ctrl/mock"`,
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
		for _, mock := range file.mocks {
			out.Decls = append(out.Decls, &dst.GenDecl{
				Tok: token.TYPE,
				Specs: []dst.Spec{
					&dst.TypeSpec{
						Name: dst.NewIdent(mock.name),
						Type: &dst.StructType{
							Fields: &dst.FieldList{
								List: []*dst.Field{
									{
										Type: &dst.SelectorExpr{
											X:   dst.NewIdent("mock"),
											Sel: dst.NewIdent("Mock"),
										},
									},
								},
							},
						},
					},
				},
				Decs: dst.GenDeclDecorations{
					NodeDecs: dst.NodeDecs{
						After: dst.EmptyLine,
					},
				},
			})
			for _, method := range mock.methods {
				if ft, ok := method.Type.(*dst.FuncType); ok {
					out.Decls = append(out.Decls, &dst.FuncDecl{
						Recv: &dst.FieldList{
							List: []*dst.Field{
								{
									Names: []*dst.Ident{
										dst.NewIdent("m"),
									},
									Type: &dst.StarExpr{
										X: dst.NewIdent(mock.name),
									},
								},
							},
						},
						Name: dst.NewIdent(method.Names[0].Name),
						Type: ft,
					})
				}
			}
		}
		if err := i.save(file.path, fset, out); err != nil {
			return fmt.Errorf("failed to format %s: %w", file.path, err)
		}
	}
	return nil
}
