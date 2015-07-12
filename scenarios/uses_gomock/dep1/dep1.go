package dep1

import (
	"fmt"

	"github.com/golang/mock/gomock"
)

var _ = gomock.Any()

func Modify(data string) error {
	return fmt.Errorf("Not Mocked!")
}
