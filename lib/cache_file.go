// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const CacheData = "_DATA_"

type CacheFile struct {
	f *os.File
	written bool
	changed bool
	h hash.Hash
	cache *Cache
	src string
	hash string
	data map[string]interface{}
}

func (c *Cache) GetFile(src string) (*CacheFile, error) {
	// TODO: we need to actually look for an existing entry in the cache

	dir := filepath.Join(c.root, "files")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, Cerr{"os.MkdirAll", err}
	}

	f, err := ioutil.TempFile(dir, "withmock-cache-")
	if err != nil {
		return nil, Cerr{"os.Create", err}
	}

	return &CacheFile{
		f: f,
		written: false,
		changed: false,
		h: sha512.New(),
		cache: c,
		src: src,
		hash: "",
		data: make(map[string]interface{}),
	}, nil
}

func (f *CacheFile) Write(p []byte) (int, error) {
	f.written = true
	return io.MultiWriter(f.f, f.h).Write(p)
}

func (f *CacheFile) Hash() string {
	return f.hash
}

func (f *CacheFile) Close() error {
	if !f.written {
		return nil
	}

	// if hash has been set, then we are already closed
	if f.hash != "" {
		return nil
	}

	f.changed = true

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

	f.data[CacheData] = f.hash

	return nil
}

func (f *CacheFile) Install(path string) error {
	if err := f.Close(); err != nil {
		return Cerr{"f.Close", err}
	}

	// Get the hash from data - as we could be installing either a new file, or
	// one entirely loaded from the cache ...
	hash, found := f.data[CacheData]
	if !found {
	}

	name := filepath.Join(f.cache.root, "files", hash.(string))

	if err := os.Link(name, path); err != nil {
		if err := os.Symlink(name, path); err != nil {
			return Cerr{"os.Symlink", err}
		}
	}

	if f.changed {
		// TODO: we need to persist f.data using gob ...
	}

	return nil
}

func (f *CacheFile) HasData() bool {
	return f.Has(CacheData)
}

func (f *CacheFile) Has(name ...string) bool {
	for _, n := range name {
		_, found := f.data[n]
		if !found {
			return false
		}
	}

	return true
}

func (f *CacheFile) Store(name string, data interface{}) {
	if name[0] == '_' {
		panic("Attempt to set private data member: " + name)
	}

	f.data[name] = data
}

func (f *CacheFile) Get(name string) interface{} {
	return f.data[name]
}

func (f *CacheFile) Lookup(name string) (interface{}, bool) {
	value, found := f.data[name]
	return value, found
}
