package code

import (
	"fmt"

	"github.com/qur/withmock/scenarios/basic/lib"

	"github.com/mxk/go-sqlite/sqlite3"
)

func TryMe() error {
	fmt.Printf("sqlite version: %s\n", sqlite3.Version())
	return lib.Wibble()
}
