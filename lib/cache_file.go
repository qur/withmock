// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"crypto/sha512"
	"io"
	"hash"
	"encoding/hex"
)

type CacheFile struct {
	f *os.File
	h hash.Hash
	cache *Cache
	path string
	hash string
}

func (c *Cache) Create(path string) (*CacheFile, error) {
	dir := filepath.Join(c.root, "files")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, Cerr{"os.MkdirAll", err}
	}

	f, err := ioutil.TempFile(dir, "withmock-cache-")
	if err != nil {
		return nil, Cerr{"os.Create", err}
	}

	return &CacheFile{f, sha512.New(), c, path, ""}, nil
}

func (f *CacheFile) Write(p []byte) (int, error) {
	return io.MultiWriter(f.f, f.h).Write(p)
}

func (f *CacheFile) Hash() string {
	return f.hash
}

func (f *CacheFile) Close() error {
	// if hash has been set, then we are already closed
	if f.hash != "" {
		return nil
	}

	if err := f.f.Close(); err != nil {
		return Cerr{"os.File.Close", err}
	}

	// TODO: should be adding size into the hash calculation ...
	f.hash = hex.EncodeToString(f.h.Sum(nil))

	name := filepath.Join(f.cache.root, "files", f.hash)

	if err := os.Rename(f.f.Name(), name); err != nil {
		return Cerr{"os.Rename", err}
	}

	if err := os.Chmod(name, 0400); err != nil {
		return Cerr{"os.Chmod", err}
	}

	if err := os.Link(name, f.path); err != nil {
		if err := os.Symlink(name, f.path); err != nil {
			return Cerr{"os.Symlink", err}
		}
	}

	return nil
}
