package stdlib

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pborman/uuid"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/zip"

	"github.com/qur/withmock/lib/env"
	"github.com/qur/withmock/lib/proxy/api"
)

type Store struct {
	url     string
	scratch string
}

func New(url, scratch string) *Store {
	return &Store{
		url:     url,
		scratch: scratch,
	}
}

var apiVersion = regexp.MustCompile(`go(\d+)(\.\d+)?\.txt`)

func (s *Store) List(ctx context.Context, mod string) ([]string, error) {
	env, err := env.GetEnv()
	if err != nil {
		return nil, err
	}
	api := filepath.Join(env["GOROOT"], "api")
	log.Printf("LIST STDLIB: %s", api)

	entries, err := os.ReadDir(api)
	if err != nil {
		return nil, err
	}

	versions := []string{}
	for _, entry := range entries {
		m := apiVersion.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		if len(m[2]) == 0 {
			m[2] = ".0"
		}
		versions = append(versions, "v"+m[1]+m[2]+".0")
	}
	semver.Sort(versions)

	return versions, nil
}

func (s *Store) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *Store) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *Store) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	version := strings.TrimSuffix(ver, ".0")
	srcURL := fmt.Sprintf("%s/go%s.src.tar.gz", s.url, version)

	resp, err := http.Get(srcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download src (%s, %s): %w", mod, ver, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, api.UnknownVersion(mod, ver)
	default:
		return nil, fmt.Errorf("unexpected http status downloading src (%s, %s): %d %s", mod, ver, resp.StatusCode, resp.Status)
	}

	scratch := filepath.Join(s.scratch, mod, ver, uuid.New())
	if err := os.MkdirAll(scratch, 0755); err != nil {
		return nil, fmt.Errorf("failed to prep scratch space (%s, %s): %w", mod, ver, err)
	}

	log.Printf("DOWNLOAD GO: %s %s -> %s", mod, ver, srcURL)

	if err := unpackTar(resp.Body, scratch); err != nil {
		return nil, fmt.Errorf("failed to unpack source tar (%s, %s): %w", mod, ver, err)
	}

	modPath := filepath.Join(scratch, "go", "src", mod)

	switch dir, err := isDir(modPath); true {
	case err != nil:
		return nil, fmt.Errorf("failed to check mod exists (%s, %s): %w", mod, ver, err)
	case dir == false:
		return nil, api.UnknownVersion(mod, ver)
	}

	if err := writeModFile(ctx, modPath, mod, version); err != nil {
		return nil, fmt.Errorf("failed to write mod file (%s, %s): %w", mod, ver, err)
	}

	pr, pw := io.Pipe()
	mv := module.Version{Path: "gowm.in/std/" + mod, Version: "v" + ver}

	go func() {
		err := zip.CreateFromDir(pw, mv, modPath)
		pw.CloseWithError(err)
	}()

	return pr, nil
}

func unpackTar(r io.Reader, base string) error {
	g, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to open gzip: %w", err)
	}

	t := tar.NewReader(g)

	for {
		h, err := t.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		path := filepath.Join(base, h.Name)

		switch h.Typeflag {
		case tar.TypeReg:
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		default:
			return fmt.Errorf("unhandled tar type: %d", h.Typeflag)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file for tar: %w", err)
		}

		if _, err := io.Copy(f, t); err != nil {
			return fmt.Errorf("failed to copy file from tar: %w", err)
		}

		f.Close()
	}
}

func isDir(path string) (bool, error) {
	s, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err == nil:
		return s.IsDir(), nil
	default:
		return false, err
	}
}

func writeModFile(ctx context.Context, dest, mod, goVersion string) error {
	log.Printf("MODFILE: %s", dest)

	mf := &modfile.File{}
	if err := mf.AddModuleStmt(mod); err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	if err := mf.AddGoStmt(goVersion); err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	data, err := mf.Format()
	if err != nil {
		return fmt.Errorf("failed to format go.mod for %s: %w", dest, err)
	}

	f, err := os.Create(filepath.Join(dest, "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to create go.mod for %s: %w", dest, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write go.mod for %s: %w", dest, err)
	}

	return nil
}
