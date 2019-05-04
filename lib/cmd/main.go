package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"

	"github.com/qur/withmock/lib/control"
)

type options struct {
	Debug bool `short:"d" long:"debug" description:"Show debug output"`
	Mock  `command:"mock" description:"Create a mock-enabled version of a package"`
	Run   `command:"run" description:"Run the given command with the mocking environment setup"`
}

func Main() int {
	opts := options{}

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash|flags.PassAfterNonOption)

	parser.CommandHandler = func(cmd flags.Commander, args []string) error {
		if !opts.Debug {
			log.SetOutput(ioutil.Discard)
		}

		if cmd == nil {
			return nil
		}

		return cmd.Execute(args)
	}

	_, err := parser.Parse()
	if flagErr, ok := err.(*flags.Error); ok && flagErr.Type == flags.ErrHelp {
		fmt.Printf("%s", flagErr)
		return 0
	} else if exitErr, ok := err.(control.ExitError); ok {
		return exitErr.ExitCode()
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}

	return 0
}
