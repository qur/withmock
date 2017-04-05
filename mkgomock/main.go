// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/qur/withmock/lib"
)

var (
	debug = flag.Bool("debug", false, "enable extra output for debugging mock genertion issues")
)

func main() {
	flag.Parse()

	if !*debug {
		// Debug not enabled, so send logging into the void
		w, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to open null output: %s", err)
			os.Exit(1)
		}
		defer w.Close()
		log.SetOutput(w)
	}

	args := flag.Args()

	srcPath, dstPath, impPath := args[1], args[2], args[3]

	cfg := &lib.MockConfig{
		MOCK:   "MOCK",
		EXPECT: "EXPECT",
	}

	_, err := lib.MakePkg(srcPath, dstPath, impPath, true, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
