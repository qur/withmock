package basic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/qur/withmock/lib/proxy/api"
)

type PrefixStripper struct {
	prefix string
	store  api.Store
}

var _ api.Store = (*PrefixStripper)(nil)

func NewPrefixStripper(prefix string, store api.Store) *PrefixStripper {
	return &PrefixStripper{
		prefix: prefix,
		store:  store,
	}
}

func (p *PrefixStripper) List(ctx context.Context, mod string) ([]string, error) {
	mod, err := p.stripPrefix(mod)
	if err != nil {
		return nil, err
	}
	l, err := p.store.List(ctx, mod)
	return l, p.fixUnknownError(err)
}

func (p *PrefixStripper) Info(ctx context.Context, mod, ver string) (*api.Info, error) {
	mod, err := p.stripPrefix(mod)
	if err != nil {
		return nil, err
	}
	i, err := p.store.Info(ctx, mod, ver)
	return i, p.fixUnknownError(err)
}

func (p *PrefixStripper) ModFile(ctx context.Context, mod, ver string) (io.Reader, error) {
	mod, err := p.stripPrefix(mod)
	if err != nil {
		return nil, err
	}
	m, err := p.store.ModFile(ctx, mod, ver)
	return m, p.fixUnknownError(err)
}

func (p *PrefixStripper) Source(ctx context.Context, mod, ver string) (io.Reader, error) {
	mod, err := p.stripPrefix(mod)
	if err != nil {
		return nil, err
	}
	s, err := p.store.Source(ctx, mod, ver)
	return s, p.fixUnknownError(err)
}

func (p *PrefixStripper) stripPrefix(mod string) (string, error) {
	if !strings.HasPrefix(mod, p.prefix) {
		return "", fmt.Errorf("module '%s' didn't have prefix '%s'", mod, p.prefix)
	}
	return mod[len(p.prefix):], nil
}

func (p *PrefixStripper) fixUnknownError(err error) error {
	if ne := api.NotExist(""); errors.As(err, &ne) {
		return api.NotExist(p.prefix + string(ne))
	}
	return err
}
