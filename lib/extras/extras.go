package extras

import (
	"bytes"
	"embed"
	"io"
	"strings"
	"text/template"
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

func InterfaceModFile(mod, ver, goVersion string, w io.Writer) error {
	t, err := template.ParseFS(content, "content/interface.mod")
	if err != nil {
		return err
	}
	return t.Execute(w, map[string]string{
		"Name":      mod,
		"GoVersion": goVersion,
		"Module":    strings.Join(strings.Split(mod, "/")[1:], "/"),
		"Version":   ver,
	})
}
