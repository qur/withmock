package dep1

import (
	"fmt"

	"github.com/qur/withmock/scenarios/issue9/dep2"
)

var Wibble = &dep2.Something{}

func Modify(data string) error {
	return fmt.Errorf("Not Mocked!")
}
