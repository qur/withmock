package stdlib

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/qur/withmock/lib/env"
	"github.com/qur/withmock/lib/proxy/api"
	"golang.org/x/mod/semver"
)

type Store struct {
	d string
}

func New(scratch string) *Store {
	return &Store{
		d: scratch,
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
	return nil, fmt.Errorf("not yet implemented")
}
