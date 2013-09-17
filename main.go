// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/qur/withmock/lib"
)

var (
	work = flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting")
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

	// First we need to figure some things out

	goRoot, err := lib.GetOutput("go", "env", "GOROOT")
	if err != nil {
		return err
	}

	goPath, err := lib.GetOutput("go", "env", "GOPATH")
	if err != nil {
		return err
	}

	pkgName, err := lib.GetOutput("go", "list", ".")
	if err != nil {
		return err
	}

	// Now we need to sort out some temporary directories to work with

	tmpDir, err := ioutil.TempDir("", "withmock")
	if err != nil {
		return err
	}
	if *work {
		fmt.Fprintf(os.Stderr, "WORK=%s\n", tmpDir)
	} else {
		defer os.RemoveAll(tmpDir)
	}

	tmpPath := filepath.Join(tmpDir, "path")

	// Now we need to figure out which packages (outside of the standard
	// library) the code we are trying to test is using (and if we want the mock
	// version or not).

	rootImports, err := lib.GetRootImports(goRoot)
	if err != nil {
		return err
	}

	imports, err := lib.GetImports(".", true)
	if err != nil {
		return err
	}

	// Now we create a new GOPATH that contains all the packages that are
	// imported by the code we are interested in (generating mocks if
	// appropriate)

	needsInstall := []string{}
	standardMocks := make(map[string]string)

	processed := make(map[string]bool)
	complete := false

	for !complete {
		complete = true
		for name, mock := range imports {
			if processed[name] {
				continue
			}
			complete = false

			processed[name] = true

			if rootImports[name] {
				// Ignore standard packages unless mocked
				if mock {
					mockName, err := lib.MockStandard(goRoot, tmpPath, name)
					if err != nil {
						return err
					}
					standardMocks[name] = mockName
					needsInstall = append(needsInstall, mockName)
				}
				continue
			}

			needsInstall = append(needsInstall, name)

			mkpkg := lib.LinkPkg
			if mock {
				mkpkg = lib.GenMockPkg
			}

			// Process the package and get it's imports
			pkgImports, err := mkpkg(goPath, tmpPath, name)
			if err != nil {
				return err
			}

			// Update imports from the package we just processed
			for name, mock := range pkgImports {
				imports[name] = imports[name] || mock
			}
		}
	}

	// Add in the gocov library, so that we can run with gocov if we want.

	_, err = lib.LinkPkg(goPath, tmpPath, "github.com/axw/gocov")
	if err != nil {
		return err
	}

	// Add the actual code that we are interested in to the GOPATH too.

	codeDest := filepath.Join(tmpPath, "src", pkgName)
	codeSrc, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	err = lib.MockImports(codeSrc, codeDest, standardMocks)
	if err != nil {
		return err
	}

	// Update the environment to use our new GOPATH, and cd into the appropriate
	// path for the code of interest.

	err = os.Setenv("GOPATH", tmpPath)
	if err != nil {
		return err
	}
	err = os.Chdir(codeDest)
	if err != nil {
		return err
	}

	// Wrap stdout and stderr with rewriters, to put the paths back to real
	// code, not our symlinks.

	stdout := lib.NewRewriter(os.Stdout, codeDest, codeSrc)
	defer stdout.Close()
	stderr := lib.NewRewriter(os.Stderr, codeDest, codeSrc)
	defer stderr.Close()

	// Now we install our generated mock code packages

	for _, name := range needsInstall {
		cmd := exec.Command("go", "install", name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
				name, err, out)
		}
	}

	// Finally we are ready to run the given command ...

	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
