// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/qur/withmock/lib"
)

var (
	work = flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting")
	gocov = flag.Bool("gocov", false, "run tests using gocov instead of go")
	verbose = flag.Bool("v", false, "add '-v' to the command run, so the tests are run in verbose mode")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] <command> [arguments]*\n",
		os.Args[0])
	fmt.Fprintf(os.Stderr, "\nRun the specified command in an environment "+
		"where imports of the package in the current directory which are "+
		"marked for mocking are replacement by automatically generated mock "+
		"versions for use with gomock.\n\n")
	fmt.Fprintf(os.Stderr, "options:\n\n")
	flag.PrintDefaults()
}

func main() {
	err := doit()

	if exit, ok := err.(*exec.ExitError); ok {
		ws := exit.Sys().(syscall.WaitStatus)
		os.Exit(ws.ExitStatus())
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func doit() error {
	// Before we get to work, parse the command line

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	// We need at least one argument

	pkgs := []string{}

	for _, arg := range args {
		list, err := lib.GetOutput("go", "list", arg)
		if err != nil {
			return err
		}
		for _, pkg := range strings.Split(list, "\n") {
			pkgs = append(pkgs, strings.TrimSpace(pkg))
		}
	}

	// First we need to create a context

	ctxt, err := lib.NewContext()
	if err != nil {
		return err
	}
	defer ctxt.Close()

	if *work {
		ctxt.KeepWork()
	}

	// Start building the command string that we will run

	command := "go"
	args = []string{"test"}
	if *verbose {
		args = append(args, "-v")
	}

	// Now we add the packages that we want to test to the context, this will
	// install the imports used by those packages (mocking them as approprite).

	for _, pkg := range pkgs {
		name, err := ctxt.AddPackage(pkg)
		if err != nil {
			return err
		}
		args = append(args, name)
	}

	// Add in the gocov library, so that we can run with gocov if we want.

	if *gocov {
		if err := ctxt.LinkPkg("github.com/axw/gocov"); err != nil {
			return err
		}
		command = "gocov"
	}

	// Finally we can run the command inside the context

	return ctxt.Run(command, args...)
}
