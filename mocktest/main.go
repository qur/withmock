// Copyright 2013 Julian Phillips.  All rights reserved.
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
	raw      = flag.Bool("raw", false, "don't rewrite the test output")
	work     = flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting")
	gocov    = flag.Bool("gocov", false, "run tests using gocov instead of go")
	verbose  = flag.Bool("v", false, "add '-v' to the command run, so the tests are run in verbose mode")
	pkgFile  = flag.String("P", "", "install extra packages listed in the given file")
	exclFile = flag.String("exclude", "", "any package listed in the given file will not be mocked, even if marked in test code.")
	cfgFile  = flag.String("c", "", "load config from the specified file")
	debug    = flag.Bool("debug", false, "enable extra output for debugging mock genertion issues")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] [package spec]*\n",
		os.Args[0])
	fmt.Fprintf(os.Stderr, "\nRun 'go test' on the specified packages in an "+
		"environment where imports of the specified packages which are "+
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

	if c, ok := err.(lib.Cerr); *debug && ok {
		fmt.Fprintf(os.Stderr, "ERROR(%s): %s\n", c.Context(), err)
		os.Exit(1)
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
			return lib.Cerr{"go list", err}
		}
		for _, pkg := range strings.Split(list, "\n") {
			pkg = strings.TrimSpace(pkg)
			if len(pkg) == 0 {
				continue
			}
			pkgs = append(pkgs, pkg)
		}
	}

	if len(pkgs) == 0 {
		fmt.Printf("no packages to test\n")
		os.Exit(1)
	}

	// First we need to create a context

	ctxt, err := lib.NewContext()
	if err != nil {
		return lib.Cerr{"NewContext", err}
	}
	defer ctxt.Close()

	if *work {
		ctxt.KeepWork()
	}

	if *raw {
		ctxt.DisableRewrite()
	}

	// Load the excluded packages file if configured

	if *exclFile != "" {
		if err := ctxt.ExcludePackagesFromFile(*exclFile); err != nil {
			return lib.Cerr{"ExcludePackagesFromFile", err}
		}
	}

	// Load the config file if specified

	if *cfgFile != "" {
		if err := ctxt.LoadConfig(*cfgFile); err != nil {
			return lib.Cerr{"LoadConfig", err}
		}
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
			return lib.Cerr{"AddPackage", err}
		}
		args = append(args, name)
	}

	// Add extra packages if configured

	if *pkgFile != "" {
		if err := ctxt.LinkPackagesFromFile(*pkgFile); err != nil {
			return lib.Cerr{"LinkPackagesFromFile", err}
		}
	}

	// Add in the gocov library, so that we can run with gocov if we want.

	if *gocov {
		if err := ctxt.LinkPackage("github.com/axw/gocov"); err != nil {
			return lib.Cerr{"LinkPackage(gocov)", err}
		}
		command = "gocov"
	}

	// Finally we can run the command inside the context

	if err := ctxt.Run(command, args...); err != nil {
		return lib.Cerr{"Run", err}
	}

	return nil
}
