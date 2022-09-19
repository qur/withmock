package mock

import (
	"context"
	"fmt"
	"go/token"
	"log"

	"github.com/dave/dst"
)

type fileInfo struct {
	pkg        *pkgInfo
	imports    []*dst.ImportSpec
	pkgs       map[string]*pkgInfo
	interfaces map[string]*interfaceInfo
}

func (fi *fileInfo) discoverInterfaces(ctx context.Context, file *dst.File) error {
	fi.interfaces = map[string]*interfaceInfo{}
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

func (fi *fileInfo) getPackages(ctx context.Context) error {
	fi.pkgs = map[string]*pkgInfo{}
	for _, imp := range fi.imports {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		path := imp.Path.Value
		log.Printf("RESOLVE IMPORT: %s", path)
		pkg, err := fi.pkg.mod.findPackage(ctx, path[1:len(path)-1])
		if err != nil {
			return err
		}
		name := pkg.name
		if imp.Name != nil {
			name = imp.Name.Name
		}
		fi.pkgs[name] = pkg
	}
	return nil
}

func (fi *fileInfo) findPackage(ctx context.Context, name string) (*pkgInfo, error) {
	if fi.pkgs == nil {
		if err := fi.getPackages(ctx); err != nil {
			return nil, err
		}
	}

	if pkg := fi.pkgs[name]; pkg != nil {
		return pkg, nil
	}

	return nil, fmt.Errorf("unknown package: %s", name)
}
