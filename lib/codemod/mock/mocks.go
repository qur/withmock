package mock

import (
	"context"
	"go/token"
	"path/filepath"

	"github.com/dave/dst"
)

func (mi *modInfo) renderMocks(ctx context.Context, dest string) error {
	for _, pkg := range mi.pkgs {
		for fileName, file := range pkg.files {
			if err := ctx.Err(); err != nil {
				// request cancelled, give up
				return err
			}

			out := &dst.File{
				Name: dst.NewIdent(pkg.name),
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

			for _, iface := range file.interfaces {
				name := "Mock" + iface.name
				out.Decls = append(out.Decls, &dst.GenDecl{
					Tok: token.TYPE,
					Specs: []dst.Spec{
						&dst.TypeSpec{
							Name: dst.NewIdent(name),
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
				for _, method := range iface.methods {
					out.Decls = append(out.Decls, method.genFunc(name))
				}
			}

			path := filepath.Join(dest, pkg.path, fileName)
			if err := save(path, mi.fset, out); err != nil {
				return err
			}
		}
	}
	return nil
}