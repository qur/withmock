package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/qur/withmock/lib/proxy/api"
)

type Dir struct {
	cache string
	s     api.Store
}

func NewDir(cache string, s api.Store) *Dir {
	return &Dir{cache: cache, s: s}
}

func (d *Dir) List(ctx context.Context, mod string) ([]string, error) {
	return d.s.List(ctx, mod)
}

func (d *Dir) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	path := filepath.Join(d.cache, mod, ver, "info.json")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		info, err := d.s.Info(ctx, mod, ver)
		if err != nil {
			return nil, err
		}
		f, err := d.createEntry(mod, ver, "info.json", nil)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(info); err != nil {
			return nil, fmt.Errorf("failed to write info cache (%s, %s): %w", mod, ver, err)
		}
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read info cache (%s, %s): %w", mod, ver, err)
	}
	defer f.Close()
	info := api.Info{}
	if err := json.NewDecoder(f).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info cache (%s, %s): %w", mod, ver, err)
	}
	return &info, nil
}

func (d *Dir) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	path := filepath.Join(d.cache, mod, ver, "go.mod")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		mf, err := d.s.ModFile(ctx, mod, ver)
		if err != nil {
			return nil, err
		}
		return d.createEntry(mod, ver, "go.mod", mf)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read mod cache (%s, %s): %w", mod, ver, err)
	}
	return f, nil
}

func (d *Dir) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	path := filepath.Join(d.cache, mod, ver, "go.zip")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		src, err := d.s.Source(ctx, mod, ver)
		if err != nil {
			return nil, err
		}
		return d.createEntry(mod, ver, "go.zip", src)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read mod cache (%s, %s): %w", mod, ver, err)
	}
	return f, nil
}

type newFile struct {
	src     io.Reader
	dst     io.WriteCloser
	tee     io.Reader
	errored bool
	temp    string
	path    string
}

var _ io.ReadCloser = (*newFile)(nil)
var _ io.Writer = (*newFile)(nil)

func (d *Dir) createEntry(mod, ver, filename string, src io.Reader) (*newFile, error) {
	dir := filepath.Join(d.cache, mod, ver)
	tempPath := filepath.Join(dir, "."+filename)
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prep cache (%s, %s): %w", mod, ver, err)
	}
	f, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache (%s, %s): %w", mod, ver, err)
	}
	return &newFile{
		src:  src,
		dst:  f,
		tee:  io.TeeReader(src, f),
		temp: tempPath,
		path: path,
	}, nil
}

func (f *newFile) Write(p []byte) (int, error) {
	return f.dst.Write(p)
}

func (f *newFile) Read(p []byte) (int, error) {
	n, err := f.tee.Read(p)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (f *newFile) Close() error {
	if closer, ok := f.src.(io.Closer); ok {
		closer.Close()
	}
	if err := f.dst.Close(); err != nil {
		return err
	}
	if !f.errored {
		return os.Rename(f.temp, f.path)
	}
	return nil
}
