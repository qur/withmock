package mock

import (
	"context"
	"fmt"
	"go/ast"
)

type pkgInfo struct {
	mod   *modInfo
	name  string
	files map[string]*fileInfo
}

func (pi *pkgInfo) resolveInterfaces(ctx context.Context, pkg *ast.Package) (int, error) {
	return 0, fmt.Errorf("not implemented")
}
