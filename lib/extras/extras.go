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
		data = bytes.Replace(data, []byte("wmqe_package_name"), []byte(pkg), -1)
	}
	return data, err
}
