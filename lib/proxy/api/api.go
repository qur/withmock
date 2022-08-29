package api

import (
	"fmt"
	"io"
	"time"
)

type Store interface {
	List(mod string) ([]string, error)
	Info(mod, ver string) (*Info, error)
	ModFile(mod, ver string) (io.Reader, error)
	Source(mod, ver string) (io.Reader, error)
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
