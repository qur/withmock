// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/qur/withmock/utils"
	"encoding/gob"
	"path/filepath"
	"io/ioutil"
)

type cacheFileDetails struct {
	Src string `json:"src"`
	Size int64 `json:"size"`
	Mode os.FileMode `json:"mode"`
	ModTime time.Time `json:"mod_time"`
	Hash string `json:"hash"`
}

func newDetails(path string) (cacheFileDetails, error) {
	st, err := os.Stat(path)
	if err != nil {
		return cacheFileDetails{}, utils.Err{"os.Stat", err}
	}

	return cacheFileDetails{
		Src: path,
		Size: st.Size(),
		Mode: st.Mode(),
		ModTime: st.ModTime(),
	}, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", utils.Err{"os.Open", err}
	}
	defer f.Close()

	h := NewCacheHash()

	log.Printf("START: calcHash")
	if _, err := io.Copy(h, f); err != nil {
		return "", utils.Err{"io.Copy", err}
	}
	hash := hex.EncodeToString(h.Sum(nil))
	log.Printf("END: calcHash")

	return hash, nil
}

func getDetails(path string) (cacheFileDetails, error) {
	d, err := newDetails(path)
	if err != nil {
		return cacheFileDetails{}, utils.Err{"newDetails", err}
	}

	d.Hash, err = hashFile(path)
	if err != nil {
		return cacheFileDetails{}, utils.Err{"hashFile", err}
	}

	return d, nil
}

func (c *Cache) loadDetails(dHash string) (string, error) {
	path := filepath.Join(c.root, "details", dHash)

	f, err := os.Open(path)
	if err != nil {
		return "", utils.Err{"os.Open", err}
	}
	defer f.Close()

	dec := gob.NewDecoder(f)

	hash := ""

	if err := dec.Decode(&hash); err != nil {
		return "", utils.Err{"gob.Decode", err}
	}

	return hash, nil
}

func (c *Cache) saveDetails(dHash, hash string) error {
	dir := filepath.Join(c.root, "details")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return utils.Err{"os.MkdirAll", err}
	}

	w, err := ioutil.TempFile(dir, "withmock-details-")
	if err != nil {
		return utils.Err{"ioutil.TempFile", err}
	}
	defer w.Close()

	enc := gob.NewEncoder(w)

	if err := enc.Encode(hash); err != nil {
		return utils.Err{"gob.Encode", err}
	}

	path := filepath.Join(c.root, "details", dHash)

	w.Close()
	if err := os.Rename(w.Name(), path); err != nil {
		return utils.Err{"os.Rename", err}
	}

	return nil
}

func (c *Cache) lookupDetails(path string) (string, error) {
	d, err := newDetails(path)
	if err != nil {
		return "", utils.Err{"newDetails", err}
	}

	h := NewCacheHash()

	enc := json.NewEncoder(h)

	if err := enc.Encode(d); err != nil {
		panic("Failed to JSON encode cacheFileKey instance: " + err.Error())
	}

	dHash := hex.EncodeToString(h.Sum(nil))

	hash, err := c.loadDetails(dHash)
	if err == nil {
		return hash, nil
	}

	hash, err = hashFile(path)
	if err != nil {
		return "", utils.Err{"hashFile", err}
	}

	if err := c.saveDetails(dHash, hash); err != nil {
		return "", utils.Err{"saveDetails", err}
	}

	return hash, nil
}
