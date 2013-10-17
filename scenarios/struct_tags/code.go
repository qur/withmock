package code

import (
	"reflect"

	"github.com/qur/withmock/scenarios/struct_tags/lib"
)

func TryMe(name string) (string, bool) {
	t := reflect.TypeOf(lib.Foo{})
	f, found := t.FieldByName(name)
	if !found {
		return "", false
	}
	return string(f.Tag), true
}
