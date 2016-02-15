package withdeps

import (
	"gopkg.in/gcfg.v1"
)

func TryMe(data string) error {
	cfg := struct {
		Section struct {
			Name string
		}
	}{}
	return gcfg.ReadStringInto(&cfg, data)
}
