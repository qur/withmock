package code

import (
	"fmt"

	"github.com/qur/withmock/scenarios/basic/lib"

	"code.google.com/p/go-sqlite/go1/sqlite3"
)

func TryMe() error {
	fmt.Printf("sqlite version: %s\n", sqlite3.Version())
	return lib.Wibble()
}
