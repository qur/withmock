package mock

import "github.com/dave/dst"

type fileInfo struct {
	mod        *modInfo
	pkg        *pkgInfo
	imports    []*dst.ImportSpec
	interfaces map[string]*interfaceInfo
}
