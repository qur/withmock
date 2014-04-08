// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"go/ast"
	"go/build"
	"log"
	"strings"
)

var (
	goos   = build.Default.GOOS
	goarch = build.Default.GOARCH
)

func goodOSArchConstraints(file *ast.File) (ok bool) {
	max := file.Package

	for _, comment := range file.Comments {
		if comment.Pos() >= max {
			break
		}

		if len(comment.List) == 0 {
			continue
		}

		line := comment.List[0].Text
		line = strings.TrimLeft(line, "/")
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "+build ") {
			continue
		}

		// Loop over lines == AND
		for _, cmt := range comment.List {
			line := cmt.Text
			line = strings.TrimLeft(line, "/")
			line = strings.TrimSpace(line)

			if len(line) == 0 {
				continue
			}

			if !strings.HasPrefix(line, "+build ") {
				log.Printf("Can't parse: '%s'", line)
				panic("Unable to parse build constraints: " + file.Name.Name)
			}

			line = strings.TrimSpace(line)[7:]

			satisfied := false

			// Loop over groups == OR
			for _, group := range strings.Split(line, " ") {
				gSatisfied := true

				// Loop over constraints == AND
				for _, constraint := range strings.Split(group, ",") {
					if constraint == goos || constraint == goarch {
						continue
					}

					if knownOS[constraint] || knownArch[constraint] {
						gSatisfied = false
					}

					if constraint == "ignore" {
						gSatisfied = false
					}
				}

				if gSatisfied {
					satisfied = true
				}
			}

			if !satisfied {
				return false
			}
		}
	}

	return true
}
