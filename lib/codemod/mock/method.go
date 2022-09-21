package mock

import (
	"go/token"

	"github.com/dave/dst"
)

type methodInfo struct {
	name      string
	signature *dst.FuncType
}

func (mi *methodInfo) genFunc(name string) *dst.FuncDecl {
	called := &dst.CallExpr{
		Fun: &dst.SelectorExpr{
			X:   dst.NewIdent("m"),
			Sel: dst.NewIdent("Called"),
		},
	}
	body := []dst.Stmt{}
	if mi.signature.Results == nil {
		body = append(body, &dst.ExprStmt{X: called})
	} else {
		results := []dst.Expr{}
		body = append(body, &dst.AssignStmt{
			Lhs: []dst.Expr{dst.NewIdent("args")},
			Tok: token.DEFINE,
			Rhs: []dst.Expr{called},
		}, &dst.ReturnStmt{Results: results})
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
	}
}
