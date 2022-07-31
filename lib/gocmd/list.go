package gocmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type Module struct {
	Path      string
	Main      bool
	Version   string
	Time      time.Time
	Indirect  bool
	Dir       string
	GoMod     string
	GoVersion string
	Replace   *Module
}

func ListMods(packages string) ([]Module, error) {
	cmd := exec.Command("go", "list", "-json", "-m", packages)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	mods, err := getMods(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return mods, nil
}

func getMods(r io.Reader) ([]Module, error) {
	mods := []Module{}
	d := json.NewDecoder(r)
	for {
		fmt.Printf(".")
		mod := Module{}
		if err := d.Decode(&mod); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mods = append(mods, mod)
	}
	fmt.Printf("\n")
	return mods, nil
}
