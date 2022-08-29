package cache

import (
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

func (d *Dir) List(mod string) ([]string, error) {
	return d.s.List(mod)
}

func (d *Dir) Info(mod, ver string) (*api.Info, error) {
	path := filepath.Join(d.cache, mod, ver, "info.json")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		info, err := d.s.Info(mod, ver)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("failed to prep info cache (%s, %s): %w", mod, ver, err)
		}
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create info cache (%s, %s): %w", mod, ver, err)
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

func (d *Dir) ModFile(mod, ver string) (io.Reader, error) {
	path := filepath.Join(d.cache, mod, ver, "go.mod")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		mf, err := d.s.ModFile(mod, ver)
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

func (d *Dir) Source(mod, ver string) (io.Reader, error) {
	path := filepath.Join(d.cache, mod, ver, "go.zip")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		src, err := d.s.Source(mod, ver)
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
	dst     io.ReadCloser
	tee     io.Reader
	errored bool
	temp    string
	path    string
}

var _ io.ReadCloser = (*newFile)(nil)

func (d *Dir) createEntry(mod, ver, filename string, src io.Reader) (*newFile, error) {
	dir := filepath.Join(d.cache, mod, ver)
	tempPath := filepath.Join(dir, "."+filename)
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to prep info cache (%s, %s): %w", mod, ver, err)
	}
	f, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create info cache (%s, %s): %w", mod, ver, err)
	}
	return &newFile{
		src:  src,
		dst:  f,
		tee:  io.TeeReader(src, f),
		temp: tempPath,
		path: path,
	}, nil
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
