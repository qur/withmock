package cmd

import (
	"fmt"
	"go/parser"
	"go/token"
)

type Mock struct {
}

func (m *Mock) Execute(args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	// fmt.Printf("modules: \n")
	// mods, err := gocmd.ListMods("all")
	// if err != nil {
	// 	return err
	// }
	// fmt.Printf("%#v\n", mods)
	// for _, mod := range mods {
	// 	if mod.Main || mod.Indirect {
	// 		continue
	// 	}
	// 	fmt.Printf("%s\n", mod.Path)
	// }
	fmt.Printf("parse: \n")
	fs := token.NewFileSet()
	for _, path := range args {
		pkgs, err := parser.ParseDir(fs, path, nil, 0)
		if err != nil {
			return err
		}
		fmt.Printf("parsed: %#v\n", pkgs)
	}
	return nil
}
