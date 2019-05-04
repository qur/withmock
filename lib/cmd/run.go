package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/qur/withmock/lib/control"
)

type Run struct{}

func (r *Run) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("No Command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() > 0 {
			return control.Exited(err, exit.ExitCode())
		}
		return err
	}
	return nil
}
