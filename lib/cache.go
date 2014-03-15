// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"os"
	"path/filepath"
)

type Cache struct {
	enabled bool
	root string
	tmpDir string
}

func NewCache(tmpDir string) *Cache {
	enabled := os.Getenv("WITHMOCK_DISABLE_CACHE") == ""

	home := os.Getenv("HOME")

	root := os.Getenv("WITHMOCK_CACHE_DIR")
	if root == "" {
		if home == "" {
			enabled = false
		}
		filepath.Join(home, ".withmock", "cache")
	}

	return &Cache{
		enabled: enabled,
		root: root,
		tmpDir: tmpDir,
	}
}
