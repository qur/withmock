// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/qur/withmock/lib"
)

func main() {
	srcPath, dstPath := os.Args[1], os.Args[2]

	err := lib.MakeMock(srcPath, dstPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
