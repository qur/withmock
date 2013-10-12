package dep1

import (
	"fmt"
	"os"
)

var Wibble *os.File

func Modify(data string) error {
	return fmt.Errorf("Not Mocked!")
}
