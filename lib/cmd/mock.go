package cmd

import (
	"fmt"

	"github.com/qur/withmock/lib/gocmd"
)

type Mock struct {
}

func (m *Mock) Execute(args []string) error {
	mods, err := gocmd.ListMods("all")
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", mods)
	for _, mod := range mods {
		if mod.Main || mod.Indirect {
			continue
		}
		fmt.Printf("%s\n", mod.Path)
	}
	return nil
}
