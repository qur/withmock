package issue8

import (
	"github.com/qur/withmock/scenarios/issue8/bug"
)

func TryMe(c chan interface{}) error {
	return bug.TryMe(c)
}
