package withdeps

import (
	"github.com/qur/withmock/scenarios/issue9/dep1"
)

func Show(data string) error {
	return dep1.Modify(data)
}
