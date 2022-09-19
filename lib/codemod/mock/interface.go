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
				iface, err := i.findInterface(ctx, name.Name, t.Sel.Name)
				if err != nil {
					return nil, err
				}
				if err := i.copyMethods(ctx, iface); err != nil {
					return nil, err
				}
			} else {
				log.Printf("    NEED ?.%s (%T)", t.Sel, t.X)
				return nil, fmt.Errorf("don't know how to resolve package for %s (?.%s): found %T", i.name, t.Sel, t.X)
			}
		case *dst.Ident:
			if t.Path != "" {
				// this is probably a type from another package
				log.Printf("    NEED %s", t)
				return nil, fmt.Errorf("don't know how to resolve interface for %s", t)
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
		default:
			return nil, fmt.Errorf("don't know how to handle interface field of type %T for %s", t, i.name)
		}
	}

	return i.methods, nil
}

func (i *interfaceInfo) findInterface(ctx context.Context, in, name string) (*interfaceInfo, error) {
	pkg, err := i.file.findPackage(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to find package for %s: %w", in, err)
	}
	if err := pkg.discoverInterfaces(ctx); err != nil {
		return nil, err
	}
	iface := pkg.interfaces[name]
	if iface == nil {
		return nil, fmt.Errorf("failed to find interface %s in %s", name, pkg.fullPath)
	}
	return iface, nil
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
