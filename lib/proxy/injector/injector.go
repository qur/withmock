package injector

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pborman/uuid"
	"github.com/qur/withmock/lib/proxy/api"
)

type Modifier interface {
	Modify(path string) error
}

type Injector struct {
	m Modifier
	d string
	s api.Store
}

var _ api.Store = (*Injector)(nil)

func NewInjector(modify Modifier, scratchDir string, s api.Store) *Injector {
	return &Injector{
		m: modify,
		d: scratchDir,
		s: s,
	}
}

func (i *Injector) List(mod string) ([]string, error) {
	return i.s.List(mod)
}

func (i *Injector) Info(mod, ver string) (*api.Info, error) {
	return i.s.Info(mod, ver)
}

func (i *Injector) ModFile(mod, ver string) (io.Reader, error) {
	return i.s.ModFile(mod, ver)
}

func (i *Injector) Source(mod, ver string) (io.Reader, error) {
	r, err := i.s.Source(mod, ver)
	if err != nil {
		return nil, err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	dir := filepath.Join(i.d, mod, ver, uuid.New())
	orig := filepath.Join(dir, "original.zip")
	src := filepath.Join(dir, "src")
	modded := filepath.Join(dir, "modded.zip")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prep zip (%s, %s): %w", mod, ver, err)
	}
	if err := save(orig, r); err != nil {
		return nil, fmt.Errorf("failed to save zip (%s, %s): %w", mod, ver, err)
	}
	headers, err := unpack(orig, src)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack zip (%s, %s): %w", mod, ver, err)
	}
	if err := i.m.Modify(src); err != nil {
		return nil, fmt.Errorf("failed to modify zip (%s, %s): %w", mod, ver, err)
	}
	if err := pack(src, modded, headers); err != nil {
		return nil, fmt.Errorf("failed to pack zip (%s, %s): %w", mod, ver, err)
	}
	return os.Open(modded)
}

func save(dest string, src io.Reader) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, src)
	return err
}

func unpack(filename, path string) ([]zip.FileHeader, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	headers := make([]zip.FileHeader, len(r.File))
	for i, f := range r.File {
		headers[i] = f.FileHeader
		if f.Mode().IsRegular() {
			if err := extractFile(path, f); err != nil {
				return nil, err
			}
		}
	}
	return headers, nil
}

func extractFile(base string, f *zip.File) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	path := filepath.Join(base, f.Name)
	if !strings.HasPrefix(path, filepath.Clean(base)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return save(path, r)
}

func pack(base, filename string, headers []zip.FileHeader) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	defer z.Close()
	for i, hdr := range headers {
		w, err := z.CreateHeader(&headers[i])
		if err != nil {
			return err
		}
		if hdr.Mode().IsRegular() {
			path := filepath.Join(base, hdr.Name)
			if err := addFile(path, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func addFile(path string, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
