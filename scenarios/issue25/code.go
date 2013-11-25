package withdeps

import (
	"code.google.com/p/gcfg"
)

func TryMe(data string) error {
	cfg := struct {
		Section struct {
			Name string
		}
	}{}
	return gcfg.ReadStringInto(&cfg, data)
}
