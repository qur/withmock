// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"hash"
	"crypto/sha512"
	"bytes"
	"encoding/binary"
)

type CacheHash struct {
	h hash.Hash
	n int
}

func NewCacheHash() *CacheHash {
	return &CacheHash{sha512.New(), 0}
}

func (h *CacheHash) Write(b []byte) (int, error) {
	n, err := h.h.Write(b)
	h.n += n
	return n, err
}

func (h *CacheHash) Sum(b []byte) []byte {
	buf := bytes.NewBuffer(b)
	binary.Write(buf, binary.BigEndian, uint64(h.n))
	buf.Write(h.h.Sum(nil))
	return buf.Bytes()
}

func (h *CacheHash) Reset() {
	h.h.Reset()
	h.n = 0
}

func (h *CacheHash) Size() int {
	return h.h.Size()
}

func (h *CacheHash) BlockSize() int {
	return h.h.BlockSize()
}
