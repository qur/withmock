package dep1

import (
	"os"
	"fmt"
)

var Wibble *os.File

func Modify(data string) error {
	return fmt.Errorf("Not Mocked!")
}
