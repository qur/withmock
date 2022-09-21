package mock

import (
	"fmt"
	"go/token"
	"strconv"

	"github.com/dave/dst"
)

type methodInfo struct {
	name      string
	signature *dst.FuncType
}

func (mi *methodInfo) genFunc(name string) *dst.FuncDecl {
	args := []dst.Expr{}
	if mi.signature.Params != nil {
		for _, arg := range mi.signature.Params.List {
			if len(arg.Names) == 0 {
				arg.Names = append(arg.Names, dst.NewIdent(fmt.Sprintf("arg_%d", len(args))))
			}
			for _, name := range arg.Names {
				if name.Name == "_" {
					name.Name = fmt.Sprintf("arg_%d", len(args))
				}
				args = append(args, dst.NewIdent(name.Name))
			}
		}
	}
	called := &dst.CallExpr{
		Fun: &dst.SelectorExpr{
			X: &dst.SelectorExpr{
				X:   dst.NewIdent("m"),
				Sel: dst.NewIdent("Mock"),
			},
			Sel: dst.NewIdent("Called"),
		},
		Args: args,
	}
	body := []dst.Stmt{}
	if mi.signature.Results == nil {
		body = append(body, &dst.ExprStmt{X: called})
	} else {
		body = append(body, mi.buildReturn(called)...)
	}
	return &dst.FuncDecl{
		Recv: &dst.FieldList{
			List: []*dst.Field{
				{
					Names: []*dst.Ident{
						dst.NewIdent("m"),
					},
					Type: &dst.StarExpr{
						X: dst.NewIdent(name),
					},
				},
			},
		},
		Name: dst.NewIdent(mi.name),
		Type: mi.signature,
		Body: &dst.BlockStmt{
			List: body,
		},
		Decs: dst.FuncDeclDecorations{
			NodeDecs: dst.NodeDecs{
				After: dst.EmptyLine,
			},
		},
	}
}

func (mi *methodInfo) buildReturn(called dst.Expr) []dst.Stmt {
	body := []dst.Stmt{}

	types := []dst.Expr{}
	for _, ret := range mi.signature.Results.List {
		if len(ret.Names) == 0 {
			types = append(types, dst.Clone(ret.Type).(dst.Expr))
		}
		for range ret.Names {
			types = append(types, dst.Clone(ret.Type).(dst.Expr))
		}
	}

	specs := []dst.Spec{}
	results := []dst.Expr{}
	for i, t := range types {
		results = append(results, dst.NewIdent(fmt.Sprintf("ret_%d", i)))
		specs = append(specs, &dst.ValueSpec{
			Names: []*dst.Ident{dst.NewIdent(fmt.Sprintf("ret_%d", i))},
			Type:  dst.Clone(t).(dst.Expr),
		})
	}

	body = append(body, &dst.AssignStmt{
		Lhs: []dst.Expr{dst.NewIdent("ret")},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{called},
	}, &dst.DeclStmt{
		Decl: &dst.GenDecl{
			Tok:   token.VAR,
			Specs: specs,
		},
	})

	for i, t := range types {
		r := &dst.IndexExpr{
			X: dst.NewIdent("ret"),
			Index: &dst.BasicLit{
				Kind:  token.INT,
				Value: strconv.FormatInt(int64(i), 10),
			},
		}
		body = append(body, &dst.IfStmt{
			Cond: &dst.BinaryExpr{
				X:  dst.Clone(r).(dst.Expr),
				Op: token.NEQ,
				Y:  dst.NewIdent("nil"),
			},
			Body: &dst.BlockStmt{
				List: []dst.Stmt{
					&dst.AssignStmt{
						Lhs: []dst.Expr{dst.NewIdent(fmt.Sprintf("ret_%d", i))},
						Tok: token.ASSIGN,
						Rhs: []dst.Expr{&dst.TypeAssertExpr{
							X:    dst.Clone(r).(dst.Expr),
							Type: dst.Clone(t).(dst.Expr),
						}},
					},
				},
			},
		})
	}

	body = append(body, &dst.ReturnStmt{
		Results: results,
	})

	return body
}
