package issue8

import (
	"github.com/qur/withmock/scenarios/issue8/bug"
)

func TryMe(c chan interface{}) error {
	return bug.TryMe(c)
}

func TryMe2(c chan<- interface{}) error {
	return bug.TryMe2(c)
}

func TryMe3(c <-chan interface{}) error {
	return bug.TryMe3(c)
}
