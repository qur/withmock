// Copyright 2011 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"bytes"
	"io"
)

type rewriter struct {
	w       io.Writer
	buf     *bytes.Buffer
	match   []byte
	replace []byte
}

func NewRewriter(w io.Writer, old, new string) *rewriter {
	return &rewriter{
		w:       w,
		buf:     &bytes.Buffer{},
		match:   []byte(old),
		replace: []byte(new),
	}
}

func (r *rewriter) flushLines() error {
	line, err := r.buf.ReadBytes('\n')
	for err == nil {
		line = bytes.Replace(line, r.match, r.replace, -1)

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
	line := bytes.Replace(r.buf.Bytes(), r.match, r.replace, -1)

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
