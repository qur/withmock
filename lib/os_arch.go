// The code in this file is copied from the Go 1.2 source code.  The code has
// been slightly modified to make it usable out of it's original context.  See
// <goroot>/src/pkg/go/build/*.go for the original source.
//
// The original source (including information on who "The Go Authors" are) can
// be downloaded from golang.org
//
// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the Go.LICENSE file.

package lib

import (
	"strings"
	"go/build"
)

// These lists needs to match the actual list in
// <goroot>/src/pkg/go/build/syslist.go - which is unfortunately private ... :(

var knownOS = map[string]bool{
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"linux":     true,
	"nacl":      true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"windows":   true,
}

var knownArch = map[string]bool{
	"386": true,
	"amd64": true,
	"amd64p32": true,
	"arm": true,
}

// goodOSArchFile returns false if the name contains a $GOOS or $GOARCH
// suffix which does not match the current system.
// The recognized name formats are:
//
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
//
func goodOSArchFile(name string, allTags map[string]bool) bool {
	ctxt := build.Default

    if dot := strings.Index(name, "."); dot != -1 {
        name = name[:dot]
    }
    l := strings.Split(name, "_")
    if n := len(l); n > 0 && l[n-1] == "test" {
        l = l[:n-1]
    }
    n := len(l)
    if n >= 2 && knownOS[l[n-2]] && knownArch[l[n-1]] {
        if allTags != nil {
            allTags[l[n-2]] = true
            allTags[l[n-1]] = true
        }
        return l[n-2] == ctxt.GOOS && l[n-1] == ctxt.GOARCH
    }
    if n >= 1 && knownOS[l[n-1]] {
        if allTags != nil {
            allTags[l[n-1]] = true
        }
        return l[n-1] == ctxt.GOOS
    }
    if n >= 1 && knownArch[l[n-1]] {
        if allTags != nil {
            allTags[l[n-1]] = true
        }
        return l[n-1] == ctxt.GOARCH
    }
    return true
}
