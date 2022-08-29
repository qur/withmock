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
			return nil, fmt.Errorf("failed to prep info cache (%s, %s): %s", mod, ver, err)
		}
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create info cache (%s, %s): %s", mod, ver, err)
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(info); err != nil {
			return nil, fmt.Errorf("failed to write info cache (%s, %s): %s", mod, ver, err)
		}
		return info, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read info cache (%s, %s): %s", mod, ver, err)
	}
	defer f.Close()
	info := api.Info{}
	if err := json.NewDecoder(f).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info cache (%s, %s): %s", mod, ver, err)
	}
	return &info, nil
}

func (d *Dir) ModFile(mod, ver string) (io.Reader, error) {
	return d.s.ModFile(mod, ver)
}

func (d *Dir) Source(mod, ver string) (io.Reader, error) {
	return d.s.Source(mod, ver)
}
