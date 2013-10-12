// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
)

type rewrite struct {
	offset  int
	content string
}

func mockFileImports(src, dst string, change map[string]string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, src, nil,
		parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return err
	}

	testFile := strings.HasSuffix(src, "_test.go")
	rewrites := []rewrite{}

	for _, decl := range file.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if g.Tok != token.IMPORT {
			continue
		}

		for _, spec := range g.Specs {
			s := spec.(*ast.ImportSpec)

			impPath := strings.Trim(s.Path.Value, "\"")
			newPath := change[impPath]

			if newPath == "" {
				// no change needed
				continue
			}

			if testFile && getMark(newPath) != testMark {
				// for test files, we only replace the import if it was marked
				// to be mocked (as the test code might want the non-mocked
				// version too), unless the mark is testMark - which means we
				// are importing the code under test, and we want to make sure
				// we get the actual code under test, not an unmodified copy.
				comment := strings.TrimSpace(s.Comment.Text())
				if strings.ToLower(comment) != "mock" {
					continue
				}
			}

			start := fset.Position(s.Path.Pos()).Offset
			rewrites = append(rewrites, rewrite{start+1, change[impPath]})
		}
	}

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	// Start by copying the complete file contents
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	// Now we go back and apply any rewrites
	for _, rw := range rewrites {
		_, err := w.Seek(int64(rw.offset), 0)
		if err != nil {
			return err
		}
		_, err = w.WriteString(rw.content)
		if err != nil {
			return err
		}
	}

	return nil
}
