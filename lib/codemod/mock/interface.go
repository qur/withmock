package mock

import "github.com/dave/dst"

type interfaceInfo struct {
	file    *fileInfo
	name    string
	fields  []*dst.Field
	methods []methodInfo
}
