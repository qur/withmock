package mock

import (
	"context"
	"go/ast"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type pkgInfo struct {
	mod        *modInfo
	name       string
	path       string
	fullPath   string
	pkg        *ast.Package
	files      map[string]*fileInfo
	interfaces map[string]*interfaceInfo
}

func (pi *pkgInfo) resolveInterfaces(ctx context.Context) error {
	for path, f := range pi.pkg.Files {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return err
		}
		in, err := decorator.DecorateFile(pi.mod.fset, f)
		if err != nil {
			return err
		}
		fi := &fileInfo{
			pkg: pi,
		}
		for _, imp := range in.Imports {
			fi.imports = append(fi.imports, dst.Clone(imp).(*dst.ImportSpec))
		}
		if err := fi.discoverInterfaces(ctx, in); err != nil {
			return err
		}
		if len(fi.interfaces) == 0 {
			// no mocks were found in file, so skip the whole file
			continue
		}
		pi.files[filepath.Base(path)] = fi
	}
	for _, file := range pi.files {
		for name, ii := range file.interfaces {
			pi.interfaces[name] = ii
		}
	}
	for _, iface := range pi.interfaces {
		if _, err := iface.getMethods(ctx); err != nil {
			return err
		}
	}
	return nil
}
