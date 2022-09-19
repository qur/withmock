package mock

import (
	"context"
	"go/token"

	"github.com/dave/dst"
)

type fileInfo struct {
	pkg        *pkgInfo
	imports    []*dst.ImportSpec
	pkgs       map[string]*pkgInfo
	interfaces map[string]*interfaceInfo
}

func (fi *fileInfo) discoverInterfaces(ctx context.Context, file *dst.File) error {
	for _, node := range file.Decls {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
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
			fields := []*dst.Field{}
			// log.Printf("TYPE: %s %#v", t.Name.Name, ift)
			for _, f := range ift.Methods.List {
				// log.Printf("METHOD: %s %T", f.Names, f.Type)
				fields = append(fields, dst.Clone(f).(*dst.Field))
			}
			fi.interfaces[t.Name.Name] = &interfaceInfo{
				file:   fi,
				name:   t.Name.Name,
				fields: fields,
			}
		}
	}
	return nil
}
