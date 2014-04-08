// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"go/ast"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/qur/withmock/utils"
)

const Data = "_DATA_"

func init() {
	gob.Register(map[string]bool{})
	registerAstTypes()
}

type CacheFile struct {
	key *CacheFileKey
	dest string
	f *os.File
	tmpName string
	written bool
	changed bool
	h hash.Hash
	cache *Cache
	hash string
	data map[string]interface{}
}

func (c *Cache) loadFile(key *CacheFileKey, dest string) (*CacheFile, error) {
	cf := &CacheFile{
		key: key,
		dest: dest,
		written: false,
		changed: false,
		h: NewCacheHash(),
		cache: c,
		hash: "",
		data: nil,
	}

	path := filepath.Join(c.root, "metadata", key.Hash())

	f, err := os.Open(path)
	if err != nil {
		return nil, utils.Err{"os.Open", err}
	}
	defer f.Close()

	dec := gob.NewDecoder(f)

	if err := dec.Decode(&cf.data); err != nil {
		return nil, utils.Err{"gob.Decode", err}
	}

	cf.hash = cf.data[Data].(string)

	return cf, nil
}

func (c *Cache) GetFile(key *CacheFileKey, dest string) (*CacheFile, error) {
	if c.enabled && !c.ignore {
		cf, err := c.loadFile(key, dest)
		if err == nil {
			return cf, nil
		}

		if !utils.IsNotExist(err) {
			return nil, utils.Err{"loadFile", err}
		}
	}

	return &CacheFile{
		key: key,
		dest: dest,
		written: false,
		changed: false,
		h: NewCacheHash(),
		cache: c,
		hash: "",
		data: make(map[string]interface{}),
	}, nil
}

func (f *CacheFile) open() error {
	if !f.cache.enabled {
		w, err := os.Create(f.dest)
		if err != nil {
			return utils.Err{"os.Create", err}
		}

		f.f = w
		f.tmpName = w.Name()

		return nil
	}

	dir := filepath.Join(f.cache.root, "files")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return utils.Err{"os.MkdirAll", err}
	}

	w, err := ioutil.TempFile(dir, "withmock-data-")
	if err != nil {
		return utils.Err{"TempFile", err}
	}

	f.f = w
	f.tmpName = w.Name()

	return nil
}

func (f *CacheFile) dump() error {
	if f.f == nil {
		return nil
	}

	if err := f.f.Close(); err != nil {
		return utils.Err{"os.File.Close", err}
	}
	f.f = nil

	if err := os.Remove(f.tmpName); err != nil {
		return utils.Err{"os.Remove", err}
	}

	return nil
}

func (f *CacheFile) Write(p []byte) (int, error) {
	if f.f == nil {
		if err := f.open(); err != nil {
			return 0, utils.Err{"cf.open", err}
		}
	}

	f.written = true
	return io.MultiWriter(f.f, f.h).Write(p)
}

func (f *CacheFile) WriteFunc(w func(string) error) error {
	f.written = true

	if !f.cache.enabled {
		if err := w(f.dest); err != nil {
			return utils.Err{"w", err}
		}

		// TODO - use os.Stat to reset written if nothing got created?

		return nil
	}

	// Currently creating a tempfile is the only way to get a filename ...

	if err := f.open(); err != nil {
		return utils.Err{"cf.open", err}
	}

	// Get rid of f.f - w is going to make the file directly

	if err := f.dump(); err != nil {
		return utils.Err{"f.dump", err}
	}

	// Call w to write the file content

	if err := w(f.tmpName); err != nil {
		return utils.Err{"w", err}
	}

	// And now we need to read it into the hash

	i, err := os.Open(f.tmpName)
	if os.IsNotExist(err) {
		// w didn't actually create a file - so reset the written flag, and we
		// are done.
		f.written = false
		return nil
	}
	if err != nil {
		return utils.Err{"os.Open", err}
	}
	defer i.Close()

	if _, err := io.Copy(f.h, i); err != nil {
		return utils.Err{"io.Copy", err}
	}

	// done

	return nil
}

func (f *CacheFile) Hash() string {
	return f.hash
}

func (f *CacheFile) Close() error {
	if !f.written {
		if err := f.dump(); err != nil {
			return utils.Err{"f.dump", err}
		}

		return nil
	}

	// if hash has been set, then we are already closed
	if f.hash != "" {
		if err := f.dump(); err != nil {
			return utils.Err{"f.dump", err}
		}

		return nil
	}

	f.changed = true

	if f.f != nil {
		if err := f.f.Close(); err != nil {
			return utils.Err{"os.File.Close", err}
		}
		f.f = nil
	}

	if f.cache.enabled {
		f.hash = hex.EncodeToString(f.h.Sum(nil))

		name := filepath.Join(f.cache.root, "files", f.hash)

		if err := os.Rename(f.tmpName, name); err != nil {
			return utils.Err{"os.Rename", err}
		}

		if err := os.Chmod(name, 0400); err != nil {
			return utils.Err{"os.Chmod", err}
		}

		f.data[Data] = f.hash
	}

	return nil
}

func (f *CacheFile) Install() error {
	if err := f.Close(); err != nil {
		return utils.Err{"f.Close", err}
	}

	if !f.cache.enabled {
		return nil
	}

	if f.written || f.HasData() {
		// Get the hash from data - as we could be installing either a new file,
		// or one entirely loaded from the cache ...
		hash, found := f.data[Data]
		if !found {
			return fmt.Errorf("Failed to get hash")
		}

		dir := filepath.Dir(f.dest)

		if err := os.MkdirAll(dir, 0700); err != nil {
			return utils.Err{"os.MkdirAll", err}
		}

		name := filepath.Join(f.cache.root, "files", hash.(string))

		// Make sure the file looks like we just wrote it
		now := time.Now()
		if err := os.Chtimes(name, now, now); err != nil {
			return utils.Err{"os.Chtimes", err}
		}

		if err := os.Link(name, f.dest); err != nil {
			if err := os.Symlink(name, f.dest); err != nil {
				return utils.Err{"os.Symlink", err}
			}
		}
	}

	if err := f.Save(); err != nil {
		return utils.Err{"f.Save", err}
	}

	return nil
}

func (f *CacheFile) Save() error {
	if !f.changed || !f.cache.enabled {
		return nil
	}

	dir := filepath.Join(f.cache.root, "metadata")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return utils.Err{"os.MkdirAll", err}
	}

	w, err := ioutil.TempFile(dir, "withmock-meta-")
	if err != nil {
		return utils.Err{"TempFile", err}
	}
	defer w.Close()

	enc := gob.NewEncoder(w)

	if err := enc.Encode(f.data); err != nil {
		return utils.Err{"gob.Encode", err}
	}

	path := filepath.Join(f.cache.root, "metadata", f.key.Hash())

	w.Close()
	if err := os.Rename(w.Name(), path); err != nil {
		return utils.Err{"os.Rename", err}
	}

	return nil
}

func (f *CacheFile) HasData() bool {
	return f.Has(Data)
}

func (f *CacheFile) Has(name ...string) bool {
	if !f.cache.enabled {
		return false
	}

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

func registerAstTypes() {
	gob.Register(&ast.ArrayType{})
	gob.Register(&ast.AssignStmt{})
	gob.Register(&ast.BadDecl{})
	gob.Register(&ast.BadExpr{})
	gob.Register(&ast.BadStmt{})
	gob.Register(&ast.BasicLit{})
	gob.Register(&ast.BinaryExpr{})
	gob.Register(&ast.BlockStmt{})
	gob.Register(&ast.BranchStmt{})
	gob.Register(&ast.CallExpr{})
	gob.Register(&ast.CaseClause{})
	gob.Register(&ast.ChanType{})
	gob.Register(&ast.CommClause{})
	gob.Register(&ast.Comment{})
	gob.Register(&ast.CommentGroup{})
	gob.Register(&ast.CompositeLit{})
	gob.Register(&ast.DeclStmt{})
	gob.Register(&ast.DeferStmt{})
	gob.Register(&ast.Ellipsis{})
	gob.Register(&ast.EmptyStmt{})
	gob.Register(&ast.ExprStmt{})
	gob.Register(&ast.Field{})
	gob.Register(&ast.FieldList{})
	gob.Register(&ast.File{})
	gob.Register(&ast.ForStmt{})
	gob.Register(&ast.FuncDecl{})
	gob.Register(&ast.FuncLit{})
	gob.Register(&ast.FuncType{})
	gob.Register(&ast.GenDecl{})
	gob.Register(&ast.GoStmt{})
	gob.Register(&ast.Ident{})
	gob.Register(&ast.IfStmt{})
	gob.Register(&ast.ImportSpec{})
	gob.Register(&ast.IncDecStmt{})
	gob.Register(&ast.IndexExpr{})
	gob.Register(&ast.InterfaceType{})
	gob.Register(&ast.KeyValueExpr{})
	gob.Register(&ast.LabeledStmt{})
	gob.Register(&ast.MapType{})
	gob.Register(&ast.Object{})
	gob.Register(&ast.Package{})
	gob.Register(&ast.ParenExpr{})
	gob.Register(&ast.RangeStmt{})
	gob.Register(&ast.ReturnStmt{})
	gob.Register(&ast.Scope{})
	gob.Register(&ast.SelectStmt{})
	gob.Register(&ast.SelectorExpr{})
	gob.Register(&ast.SendStmt{})
	gob.Register(&ast.SliceExpr{})
	gob.Register(&ast.StarExpr{})
	gob.Register(&ast.StructType{})
	gob.Register(&ast.SwitchStmt{})
	gob.Register(&ast.TypeAssertExpr{})
	gob.Register(&ast.TypeSpec{})
	gob.Register(&ast.TypeSwitchStmt{})
	gob.Register(&ast.UnaryExpr{})
	gob.Register(&ast.ValueSpec{})
}
