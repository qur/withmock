package withdeps

import (
	"github.com/qur/withmock/scenarios/uses_gomock/dep1"
)

func Show(data string) error {
	return dep1.Modify(data)
}
