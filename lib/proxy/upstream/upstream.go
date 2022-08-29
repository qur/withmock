package upstream

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/qur/withmock/lib/proxy/api"
)

type Store struct {
	url string
}

func (s *Store) List(mod string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/@v/list", s.url, mod)
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, api.UnknownMod(mod)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream failed (%d): %s", r.StatusCode, r.Status)
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func (s *Store) Info(mod, ver string) (*api.Info, error) {
	url := fmt.Sprintf("%s/%s/@v/v%s.info", s.url, mod, ver)
	r, err := http.Get(url)
	log.Printf("INFO (%s, %s): %s %s", mod, ver, r.Status, err)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, api.UnknownMod(mod)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream failed (%d): %s", r.StatusCode, r.Status)
	}
	defer r.Body.Close()
	info := api.Info{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %s", err)
	}
	return &info, nil
}

func (s *Store) ModFile(mod, ver string) (io.Reader, error) {
	url := fmt.Sprintf("%s/%s/@v/v%s.mod", s.url, mod, ver)
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, api.UnknownMod(mod)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream failed (%d): %s", r.StatusCode, r.Status)
	}
	return r.Body, nil
}

func (s *Store) Source(mod, ver string) (io.Reader, error) {
	url := fmt.Sprintf("%s/%s/@v/v%s.zip", s.url, mod, ver)
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, api.UnknownMod(mod)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream failed (%d): %s", r.StatusCode, r.Status)
	}
	return r.Body, nil
}

func NewStore(url string) *Store {
	return &Store{url: url}
}
