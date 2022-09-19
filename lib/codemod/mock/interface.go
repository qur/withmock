package mock

import (
	"context"
	"fmt"
	"log"

	"github.com/dave/dst"
)

type interfaceInfo struct {
	file    *fileInfo
	name    string
	fields  []*dst.Field
	methods []methodInfo
}

func (i *interfaceInfo) getMethods(ctx context.Context) ([]methodInfo, error) {
	if i.methods != nil {
		return i.methods, nil
	}

	for _, field := range i.fields {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return nil, err
		}

		switch t := field.Type.(type) {
		case *dst.SelectorExpr:
			// this is probably a type from another package
			if name, ok := t.X.(*dst.Ident); ok {
				log.Printf("    NEED %s.%s", name, t.Sel)
				pkg, err := i.file.findPackage(ctx, name.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to find package for %s: %w", name.Name, err)
				}
				if err := pkg.resolveInterfaces(ctx); err != nil {
					return nil, err
				}
				iface := pkg.interfaces[t.Sel.Name]
				if iface == nil {
					return nil, fmt.Errorf("failed to find interface %s in %s", t.Sel.Name, pkg.fullPath)
				}
				if err := i.copyMethods(ctx, iface); err != nil {
					return nil, err
				}
			}
		case *dst.Ident:
			if t.Path != "" {
				// this is probably a type from another package
				log.Printf("    NEED %s", t)
				continue
			}
			// this is probably a type from the same package?
			embedded, found := i.file.pkg.interfaces[t.Name]
			if !found {
				return nil, fmt.Errorf("reference to unknown type: %s", t.Name)
			}
			if err := i.copyMethods(ctx, embedded); err != nil {
				return nil, err
			}
		case *dst.FuncType:
			// this is already a method
			i.methods = append(i.methods, methodInfo{
				name:      field.Names[0].Name,
				signature: t,
			})
		}
	}

	return i.methods, nil
}

func (i *interfaceInfo) copyMethods(ctx context.Context, other *interfaceInfo) error {
	methods, err := other.getMethods(ctx)
	if err != nil {
		return err
	}
	for _, m := range methods {
		i.methods = append(i.methods, methodInfo{
			name:      m.name,
			signature: dst.Clone(m.signature).(*dst.FuncType),
		})
	}
	return nil
}
