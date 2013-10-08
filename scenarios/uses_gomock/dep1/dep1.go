package dep1

import (
	"fmt"

	"code.google.com/p/gomock/gomock"
)

var _ = gomock.Any()

func Modify(data string) error {
	return fmt.Errorf("Not Mocked!")
}
