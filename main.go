// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/qur/withmock/lib"
)

var (
	gocov = flag.Bool("gocov", false, "install gocov package into temporary GOPATH")
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

	// We need at least one argument

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	// First we need to create a context

	ctxt, err := lib.NewContext()
	if err != nil {
		return err
	}
	defer ctxt.Close()

	// Now we add the packages that we want to test to the context, this will
	// install the imports used by those packages (mocking them as approprite).

	if err := ctxt.AddPackage("."); err != nil {
		return err
	}

	// Add in the gocov library, so that we can run with gocov if we want.

	if flag.Arg(0) == "gocov" || *gocov {
		if err := ctxt.LinkPkg("github.com/axw/gocov"); err != nil {
			return err
		}
	}

	// Finally we can run the command inside the context

	return ctxt.Run(flag.Arg(0), flag.Args()[1:]...)
}
