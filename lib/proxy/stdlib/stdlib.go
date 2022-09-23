package stdlib

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pborman/uuid"
	"golang.org/x/mod/semver"

	"github.com/qur/withmock/lib/env"
	"github.com/qur/withmock/lib/proxy/api"
)

type Store struct {
	scratch string
}

func New(scratch string) *Store {
	return &Store{
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
	srcURL := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", version)

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

	log.Printf("DOWNLOAD GO: %s", srcURL)

	if err := unpackTar(resp.Body, scratch); err != nil {
		return nil, fmt.Errorf("failed to unpack source tar (%s, %s): %w", mod, ver, err)
	}

	return nil, fmt.Errorf("not yet implemented")
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
