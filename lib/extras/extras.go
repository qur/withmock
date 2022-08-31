package extras

import (
	"bytes"
	"embed"
)

//go:embed content
var content embed.FS

func Controller(pkg string) ([]byte, error) {
	data, err := content.ReadFile("content/controller.go")
	if err == nil {
		data = bytes.Replace(data, []byte("package wmqe_package_name"), []byte("package "+pkg), 1)
	}
	return data, err
}
