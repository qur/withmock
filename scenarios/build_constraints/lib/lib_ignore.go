// This comment is a multiline comment
// which comes before the build constraint

// +build ignore

package lib

import (
	"fmt"
)

func Wibble() error {
	return fmt.Errorf("Not Mocked!")
}
