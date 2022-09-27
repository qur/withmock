package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

type Store interface {
	List(ctx context.Context, mod string) ([]string, error)
	Info(ctx context.Context, mod, ver string) (*Info, error)
	ModFile(ctx context.Context, mod, ver string) (io.Reader, error)
	Source(ctx context.Context, mod, ver string) (io.Reader, error)
}

type Info struct {
	Version string
	Time    time.Time
}

type NotExist string

func UnknownMod(mod string) NotExist {
	return NotExist(mod)
}

func UnknownVersion(mod, ver string) NotExist {
	return NotExist(fmt.Sprintf("%s@v%s", mod, ver))
}

func (n NotExist) Error() string {
	return fmt.Sprintf("not found: %s", string(n))
}

var ErrModFromSource = errors.New("mod should be extracted from source")
