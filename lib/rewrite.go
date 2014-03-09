// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"io"
	"os"
)

type rewriter struct {
	w        io.Writer
	buf      *bytes.Buffer
	rewrites []rw
}

type rw struct {
	match, replace []byte
}

func NewRewriter(w io.Writer) *rewriter {
	return &rewriter{
		w:   w,
		buf: &bytes.Buffer{},
	}
}

func (r *rewriter) Rewrite(src, dst string) {
	r.rewrites = append(r.rewrites, rw{[]byte(src), []byte(dst)})
}

func (r *rewriter) flushLines() error {
	line, err := r.buf.ReadBytes('\n')
	for err == nil {
		for _, rw := range r.rewrites {
			line = bytes.Replace(line, rw.match, rw.replace, -1)
		}

		_, err = r.w.Write(line)
		if err != nil {
			return err
		}

		line, err = r.buf.ReadBytes('\n')
	}

	// Rebuild the buffer without any data that we have processed, but including
	// any data we have read out, but not yet processed.
	buf := bytes.NewBuffer(line)
	buf.Write(r.buf.Bytes())
	r.buf = buf

	if err != io.EOF {
		return err
	}

	return nil
}

func (r *rewriter) flush() error {
	if r.buf.Len() == 0 {
		return nil
	}

	line := r.buf.Bytes()
	for _, rw := range r.rewrites {
		line = bytes.Replace(line, rw.match, rw.replace, -1)
	}

	_, err := r.w.Write(line)
	if err != nil {
		return err
	}

	r.buf.Reset()

	return nil
}

func (r *rewriter) Write(p []byte) (int, error) {
	r.buf.Write(p)
	return len(p), r.flushLines()
}

func (r *rewriter) Close() error {
	return r.flush()
}

func (r *rewriter) Change(w io.Writer) error {
	err := r.Close()
	if err != nil {
		return err
	}

	r.w = w
	return nil
}

func (rw *rewriter) Copy(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return Cerr{"os.Open", err}
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return Cerr{"os.Create", err}
	}
	defer w.Close()

	err = rw.Change(w)
	if err != nil {
		return Cerr{"rw.Change", err}
	}
	defer rw.Close()

	_, err = io.Copy(rw, r)
	if err != nil {
		return Cerr{"io.Copy", err}
	}

	return nil
}
