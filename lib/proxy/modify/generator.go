package modify

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pborman/uuid"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"

	"github.com/qur/withmock/lib/proxy/api"
)

type Generator interface {
	GenSource(ctx context.Context, mod, ver, zipfile, src, dest string) error
}

type InterfaceGenerator struct {
	g Generator
	d string
	s api.Store
}

var _ api.Store = (*InterfaceGenerator)(nil)

func NewInterfaceGenerator(generate Generator, scratchDir string, s api.Store) *InterfaceGenerator {
	return &InterfaceGenerator{
		g: generate,
		d: scratchDir,
		s: s,
	}
}

func (i *InterfaceGenerator) List(ctx context.Context, mod string) ([]string, error) {
	return i.s.List(ctx, mod)
}

func (i *InterfaceGenerator) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	return i.s.Info(ctx, mod, ver)
}

func (i *InterfaceGenerator) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	return nil, api.ErrModFromSource
}

func (i *InterfaceGenerator) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	r, err := i.s.Source(ctx, mod, ver)
	if err != nil {
		return nil, err
	}
	if closer, ok := r.(io.Closer); ok {
		defer closer.Close()
	}
	dir := filepath.Join(i.d, mod, ver, uuid.New())
	log.Printf("GENERATE SOURCE: %s %s -> %s", mod, ver, dir)
	input := filepath.Join(dir, "source.zip")
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dst")
	output := filepath.Join(dir, "interface.zip")
	mv := module.Version{Path: mod, Version: "v" + ver}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prep zip (%s, %s): %w", mod, ver, err)
	}
	if err := save(input, r); err != nil {
		return nil, fmt.Errorf("failed to save zip (%s, %s): %w", mod, ver, err)
	}
	if err := i.g.GenSource(ctx, mod, ver, input, src, dest); err != nil {
		return nil, fmt.Errorf("failed to modify zip (%s, %s): %w", mod, ver, err)
	}
	f, err := os.Create(output)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip (%s, %s): %w", mod, ver, err)
	}
	if err := zip.CreateFromDir(f, mv, dest); err != nil {
		return nil, fmt.Errorf("failed to write zip (%s, %s): %w", mod, ver, err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to seek mod file (%s, %s): %w", mod, ver, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("failed to clean zip (%s, %s): %w", mod, ver, err)
	}
	return f, nil
}
