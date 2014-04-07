// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"os"
	"path/filepath"

	"github.com/qur/withmock/config"
	"github.com/qur/withmock/utils"
)

type Cache struct {
	enabled bool
	root string
	goRoot string
	goVersion string
	self string
	cfg *config.MockConfig
}

var self string

func init() {
	hash, err := hashFile("/proc/self/exe")
	if err != nil {
		panic("Failed to generate key from binary: " + err.Error())
	}
	self = hash
}

func OpenCache(goRoot, goVersion string, cfg *config.MockConfig) (*Cache, error) {
	enabled := os.Getenv("WITHMOCK_DISABLE_CACHE") == ""

	home := os.Getenv("HOME")

	root := os.Getenv("WITHMOCK_CACHE_DIR")
	if root == "" {
		if home == "" {
			enabled = false
		}
		root = filepath.Join(home, ".withmock", "cache")
	}

	if enabled {
		if err := os.MkdirAll(root, 0700); err != nil {
			return nil, utils.Err{"os.MkdirAll", err}
		}
	}

	return &Cache{
		enabled: enabled,
		root: root,
		goRoot: goRoot,
		goVersion: goVersion,
		self: self,
		cfg: cfg,
	}, nil
}
