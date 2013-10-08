package code

import (
	"github.com/qur/withmock/scenarios/excludes/lib"
)

func TryMe() string {
	foo := lib.NewFoo()
	return foo.EXPECT()
}
