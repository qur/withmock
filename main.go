// Copyright 2013 Julian Phillips.  All rights reserved.
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
	"log"
)

var (
	raw      = flag.Bool("raw", false, "don't rewrite the command output")
	work     = flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting")
	gocov    = flag.Bool("gocov", false, "install gocov package into temporary GOPATH")
	pkgFile  = flag.String("P", "", "install extra packages listed in the given file")
	exclFile = flag.String("exclude", "", "any package listed in the given file will not be mocked, even if marked in test code.")
	cfgFile  = flag.String("c", "", "load config from the specified file")
	debug    = flag.Bool("debug", false, "enable extra output for debugging mock genertion issues")
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
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Printf("START: overall")

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

	if *work {
		ctxt.KeepWork()
	}

	if *raw {
		ctxt.DisableRewrite()
	}

	// Load the excluded packages file if configured

	if *exclFile != "" {
		if err := ctxt.ExcludePackagesFromFile(*exclFile); err != nil {
			return err
		}
	}

	// Load the config file if specified

	if *cfgFile != "" {
		if err := ctxt.LoadConfig(*cfgFile); err != nil {
			return err
		}
	}

	// Now we add the package that we want to test to the context, this will
	// install the imports used by that package (mocking them as approprite).

	pkg, err := lib.GetOutput("go", "list", ".")
	if err != nil {
		return err
	}

	log.Printf("START: AddPackage")
	testPkg, err := ctxt.AddPackage(pkg)
	if err != nil {
		return err
	}
	log.Printf("END: AddPackage")

	// Add extra packages if configured

	log.Printf("START: LinkPkg")
	if *pkgFile != "" {
		if err := ctxt.LinkPackagesFromFile(*pkgFile); err != nil {
			return err
		}
	}
	log.Printf("END: LinkPkg")

	// Add in the gocov library, so that we can run with gocov if we want.

	log.Printf("START: gocov")
	if flag.Arg(0) == "gocov" || *gocov {
		if err := ctxt.LinkPackage("github.com/axw/gocov"); err != nil {
			return err
		}
	}
	log.Printf("END: gocov")

	// Finally we can chdir into the test code, and run the command inside the
	// context

	if err := ctxt.Chdir(testPkg); err != nil {
		return err
	}

	log.Printf("START: Run")
	if err := ctxt.Run(flag.Arg(0), flag.Args()[1:]...); err != nil {
		return lib.Cerr{"Run", err}
	}
	log.Printf("END: Run")

	log.Printf("END: overall")

	return nil
}
