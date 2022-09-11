package modify

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pborman/uuid"
	"github.com/qur/withmock/lib/proxy/api"
)

type Modifier interface {
	Modify(ctx context.Context, mod, ver, path string) ([]string, error)
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

func (i *Injector) List(ctx context.Context, mod string) ([]string, error) {
	return i.s.List(ctx, mod)
}

func (i *Injector) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	return i.s.Info(ctx, mod, ver)
}

func (i *Injector) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	return i.s.ModFile(ctx, mod, ver)
}

func (i *Injector) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	r, err := i.s.Source(ctx, mod, ver)
	if err != nil {
		return nil, err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	dir := filepath.Join(i.d, mod, ver, uuid.New())
	log.Printf("INJECT: %s %s -> %s", mod, ver, dir)
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
	extraFiles, err := i.m.Modify(ctx, mod, ver, src)
	if err != nil {
		return nil, fmt.Errorf("failed to modify zip (%s, %s): %w", mod, ver, err)
	}
	headers = expandHeaders(headers, extraFiles)
	if err := pack(src, modded, headers); err != nil {
		return nil, fmt.Errorf("failed to pack zip (%s, %s): %w", mod, ver, err)
	}
	f, err := os.Open(modded)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip (%s, %s): %w", mod, ver, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("failed to clean zip (%s, %s): %w", mod, ver, err)
	}
	return f, nil
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

func expandHeaders(headers []zip.FileHeader, extraFiles []string) []zip.FileHeader {
	for _, extra := range extraFiles {
		headers = append(headers, zip.FileHeader{
			Name:   extra,
			Method: zip.Deflate,
		})
	}
	return headers
}
