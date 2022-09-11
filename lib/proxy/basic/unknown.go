package basic

import (
	"context"
	"io"
	"log"

	"github.com/qur/withmock/lib/proxy/api"
)

type Unknown struct{}

var _ api.Store = (*Unknown)(nil)

func NewUnknown() *Unknown {
	return &Unknown{}
}

func (*Unknown) List(_ context.Context, mod string) ([]string, error) {
	log.Printf("UNKNOWN LIST: %s", mod)
	return nil, api.UnknownMod(mod)
}

func (*Unknown) Info(_ context.Context, mod, ver string) (*api.Info, error) {
	log.Printf("UNKNOWN INFO: %s@v%s", mod, ver)
	return nil, api.UnknownVersion(mod, ver)
}

func (*Unknown) ModFile(_ context.Context, mod, ver string) (io.Reader, error) {
	log.Printf("UNKNOWN MOD: %s@v%s", mod, ver)
	return nil, api.UnknownVersion(mod, ver)
}

func (*Unknown) Source(_ context.Context, mod, ver string) (io.Reader, error) {
	log.Printf("UNKNOWN SOURCE: %s@v%s", mod, ver)
	return nil, api.UnknownVersion(mod, ver)
}
