package mock

import (
	"context"
	"fmt"
	"go/ast"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type pkgInfo struct {
	mod        *modInfo
	name       string
	files      map[string]*fileInfo
	interfaces map[string]*interfaceInfo
}

func (pi *pkgInfo) resolveInterfaces(ctx context.Context, pkg *ast.Package) (int, error) {
	count := 0
	for path, f := range pkg.Files {
		if err := ctx.Err(); err != nil {
			// request cancelled, give up
			return 0, err
		}
		in, err := decorator.DecorateFile(pi.mod.fset, f)
		if err != nil {
			return 0, err
		}
		fi := &fileInfo{
			pkg:        pi,
			imports:    []*dst.ImportSpec{},
			pkgs:       map[string]*pkgInfo{},
			interfaces: map[string]*interfaceInfo{},
		}
		for _, imp := range in.Imports {
			fi.imports = append(fi.imports, dst.Clone(imp).(*dst.ImportSpec))
		}
		if err := fi.discoverInterfaces(ctx, in); err != nil {
			return 0, err
		}
		if len(fi.interfaces) == 0 {
			// no mocks were found in file, so skip the whole file
			continue
		}
		rel, err := filepath.Rel(pi.mod.src, path)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve extra path %s: %s", path, err)
		}
		pi.files[rel] = fi
		count += len(fi.interfaces)
	}
	for _, file := range pi.files {
		for name, ii := range file.interfaces {
			pi.interfaces[name] = ii
		}
	}
	return count, nil
}
