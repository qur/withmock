package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/pborman/uuid"
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

func (d *Dir) getInfo(ctx context.Context, mod, ver string) (io.Reader, error) {
	info, err := d.s.Info(ctx, mod, ver)
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	go func() {
		err := json.NewEncoder(w).Encode(info)
		w.CloseWithError(err)
	}()
	return r, nil
}

func (d *Dir) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	r, err := d.entry(ctx, mod, ver, "info.json", d.getInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to read info cache (%s, %s): %w", mod, ver, err)
	}
	defer r.Close()
	info := api.Info{}
	if err := json.NewDecoder(r).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode info cache (%s, %s): %w", mod, ver, err)
	}
	return &info, nil
}

func (d *Dir) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	r, err := d.entry(ctx, mod, ver, "go.mod", d.getModFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mod cache (%s, %s): %w", mod, ver, err)
	}
	return r, nil
}

func (d *Dir) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	r, err := d.entry(ctx, mod, ver, "go.zip", d.s.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to read source cache (%s, %s): %w", mod, ver, err)
	}
	return r, nil
}

func (d *Dir) entry(ctx context.Context, mod, ver, name string, download func(context.Context, string, string) (io.Reader, error)) (io.ReadCloser, error) {
	path := filepath.Join(d.cache, mod, ver, name)
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("CACHE MISS: %s %s -> %s", mod, ver, path)
		src, err := download(ctx, mod, ver)
		if err != nil {
			return nil, err
		}
		return d.createEntry(mod, ver, name, src)
	}
	if err != nil {
		return nil, err
	}
	log.Printf("CACHE HIT: %s %s -> %s", mod, ver, path)
	return f, nil
}

func (d *Dir) getModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	mf, err := d.s.ModFile(ctx, mod, ver)
	if !errors.Is(err, api.ErrModFromSource) {
		return mf, err
	}
	src, err := d.Source(ctx, mod, ver)
	if err != nil {
		return nil, err
	}
	return extractModFile(src, d.cache, mod, ver)
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
	tempPath := filepath.Join(dir, "."+uuid.New()+"."+filename)
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
	n, err := f.dst.Write(p)
	if err != nil {
		f.errored = true
	}
	return n, err
}

func (f *newFile) Read(p []byte) (int, error) {
	n, err := f.tee.Read(p)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			f.errored = true
		}
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
	os.Remove(f.temp)
	return nil
}
