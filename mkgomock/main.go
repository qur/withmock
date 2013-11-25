// Copyright 2013 Julian Phillips.  All rights reserved.
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

	cfg := &lib.MockConfig{
		MOCK:   "MOCK",
		EXPECT: "EXPECT",
	}

	_, err := lib.MakePkg(srcPath, dstPath, true, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
