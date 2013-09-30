package withdeps

import (
	"github.com/qur/withmock/scenarios/issue10/dep1"
)

func Show(data string) error {
	return dep1.Modify(data)
}
