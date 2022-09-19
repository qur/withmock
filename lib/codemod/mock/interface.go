package mock

type interfaceInfo struct {
	mod     *modInfo
	pkg     *pkgInfo
	file    *fileInfo
	name    string
	methods []methodInfo
}
