// Copyright 2013 Julian Phillips.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Context struct {
	goPath string
	goRoot string

	tmpPath  string
	origPath string

	tmpDir    string
	removeTmp bool

	stdlibImports map[string]bool
	imports       []string

	excludes map[string]bool

	processed      map[string]bool
	importRewrites map[string]string

	doRewrite bool

	code []codeLoc

	cfg *Config

	cache *Cache
	packages map[string]Package
}

type codeLoc struct {
	src, dst string
}

func getTmpPath(tmpDir string) string {
	return filepath.Join(tmpDir, "path")
}

func NewContext() (*Context, error) {
	// First we need to figure some things out

	goRoot, err := GetOutput("go", "env", "GOROOT")
	if err != nil {
		return nil, err
	}

	goPath, err := GetOutput("go", "env", "GOPATH")
	if err != nil {
		return nil, err
	}

	stdlibImports, err := getStdlibImports(goRoot)
	if err != nil {
		return nil, err
	}

	// Now we need to sort out some temporary directories to work with

	tmpDir, err := ioutil.TempDir("", "withmock")
	if err != nil {
		return nil, err
	}

	// Setup a cache for our build area

	cache := NewCache(tmpDir)

	// Build and return the context

	return &Context{
		goPath:         goPath,
		goRoot:         goRoot,
		origPath:       os.Getenv("GOPATH"),
		tmpPath:        getTmpPath(tmpDir),
		tmpDir:         tmpDir,
		stdlibImports:  stdlibImports,
		removeTmp:      true,
		processed:      make(map[string]bool),
		importRewrites: make(map[string]string),
		doRewrite:      true,
		cfg:            &Config{},
		cache:          cache,
		packages:       make(map[string]Package),
		// create excludes already including gomock, as we can't mock it.
		excludes: map[string]bool{"code.google.com/p/gomock/gomock": true},
	}, nil
}

func (c *Context) KeepWork() {
	if c.removeTmp {
		fmt.Fprintf(os.Stderr, "WORK=%s\n", c.tmpDir)
		c.removeTmp = false
	}
}

func (c *Context) DisableRewrite() {
	c.doRewrite = false
}

func (c *Context) Close() error {
	if c.removeTmp {
		if err := os.RemoveAll(c.tmpDir); err != nil {
			return err
		}
	}

	if err := os.Setenv("GOPATH", c.origPath); err != nil {
		return err
	}

	return nil
}

func (c *Context) LoadConfig(path string) (err error) {
	c.cfg, err = ReadConfig(path)
	return
}

func (c *Context) insideCommand(command string, args ...string) *exec.Cmd {
	env := os.Environ()

	// remove any current GOPATH from the environment
	for i := range env {
		if strings.HasPrefix(env[i], "GOPATH=") {
			env[i] = "__IGNORE="
		}
	}

	// Setup the environment variables that we want
	env = append(env, "GOPATH=" + c.tmpPath)
	env = append(env, "ORIG_GOPATH=" + c.origPath)

	cmd := exec.Command(command, args...)
	cmd.Env = env
	return cmd
}

func (c *Context) installPackages() error {
	for _, pkg := range c.packages {
		if c.stdlibImports[pkg.Label()] {
			// stdlib imports don't need installing
			continue
		}

		if err := pkg.Install(); err != nil {
			return Cerr{"pkg.Install", err}
		}
	}

	return nil
}

func (c *Context) Chdir(pkg string) error {
	path := filepath.Join(c.tmpPath, "src", pkg)

	if err := os.Chdir(path); err != nil {
		return err
	}

	return nil
}

const (
	importNormal importMode = iota
	importMock
	importReplace
)

type importMode int
type importCfg struct {
	mode importMode
	path string
}
type importSet map[string]importCfg

func (i importCfg) IsMock() bool {
	return i.mode == importMock
}

func (i importCfg) IsReplace() bool {
	return i.mode == importReplace
}

func (s importSet) Set(path string, mode importMode, path2 string) error {
	i := s[path]

	if mode != importNormal {
		if i.mode != importNormal && i.mode != mode {
			return fmt.Errorf("Cannot change mode from %s to %s", i.mode, mode)
		}

		i.mode = mode
		i.path = path2
	}

	s[path] = i
	return nil
}

func (c *Context) wantToProcess(mockAllowed bool, imports importSet) map[string]string {
	names := make(map[string]string)

	for name, cfg := range imports {
		label := markImport(name, normalMark)
		if cfg.IsMock() && mockAllowed && c.stdlibImports[name] {
			label = markImport(name, mockMark)
		}
		names[name] = label

		c.processed[label] = c.processed[label] || false

		if strings.HasSuffix(label, "/_mocks_") {
			// Special mocks package that we don't want to process
			c.processed[label] = true
		}
	}

	// remove nop rewites from the names map, and add real ones to
	// c.importRewrites so they get added to the output rewrite rules.
	for orig, marked := range names {
		if orig == marked {
			delete(names, orig)
			continue
		}
		c.importRewrites[marked] = orig
	}

	return names
}

func (c *Context) installImports(imports importSet) (map[string]string, error) {
	// Start by updating processed to include anything in imports we haven't
	// seen before, this also gives us the name rewrite map we need to return

	names := c.wantToProcess(true, imports)

	// Now we update our GOPATH until it inclues all of the packages needed to
	// satisfy the dependency chain created by adding imports to the list of
	// packages that need to be installed.  This has to take into account the
	// potential desire to have the plain, mocked and test versions of the same
	// package in GOPATH at the same time ...

	complete := false

	for !complete {
		complete = true
		for label, done := range c.processed {
			if done {
				continue
			}

			complete = false
			c.processed[label] = true

			name := label
			mock := imports[name].IsMock()

			if n, found := c.importRewrites[label]; found {
				name = n
				mock = true
			}

			if imports[name].IsReplace() {
				// Install the requested package in place of the
				// package that the code thinks it wants.
				srcPath := imports[name].path
				pkgImports, err := ReplacePkg(c.goPath, c.tmpPath, srcPath, label)
				if err != nil {
					return nil, Cerr{"ReplacePkg", err}
				}

				// Update imports from the package we just processed, but it
				// can only add actual packages, not mocks
				c.wantToProcess(false, pkgImports)
			
				continue
			}

			if c.stdlibImports[name] && !mock {
				// Ignore standard packages that we aren't mocking
				continue
			}

			pkg, err := c.getPkg(name, label)
			if err != nil {
				return nil, Cerr{"context.getPkg", err}
			}
			pkg.InstallAs(label)

			cfg := c.cfg.Mock(name)

			if c.excludes[name] {
				// this package has been specifically excluded from mocking, so
				// we just link it, even if mocked is indicated.
				if _, err := pkg.Link(); err != nil {
					return nil, Cerr{"pkg.Link", err}
				}
				continue
			}

			if c.stdlibImports[name] {
				// We already checked earlier for unmocked stdlib, so
				// this is mocked stdlib
				err := MockStandard(c.goRoot, c.tmpPath, name, cfg)
				if err != nil {
					return nil, Cerr{"MockStandard", err}
				}
				continue
			}

			// Process the package and get it's imports
			pkgImports, err := pkg.Gen(mock, cfg)
			if err != nil {
				return nil, Cerr{"GenPkg", err}
			}

			// Update imports from the package we just processed, but it can
			// only add actual packages, not mocks
			c.wantToProcess(false, pkgImports)
		}
	}

	return names, nil
}

func (c *Context) getPkg(pkgName, label string) (Package, error) {
	pkg, found := c.packages[label]
	if found {
		return pkg, nil
	}

	pkg, err := c.cache.Fetch(pkgName)
	if err != nil {
		return nil, Cerr{"cache.Fetch", err}
	}

	if pkg == nil {
		pkg, err = NewPackage(pkgName, label, c.tmpDir, c.goPath)
		if err != nil {
			return nil, Cerr{"NewPackage", err}
		}
	}

	c.packages[label] = pkg

	return pkg, nil
}

func (c *Context) LinkPackage(pkg string) error {
	_, err := LinkPkg(c.goPath, c.tmpPath, pkg)
	return err
}

func (c *Context) AddPackage(pkgName string) (string, error) {
	pkg, err := c.getPkg(pkgName, markImport(pkgName, testMark))
	if err != nil {
		return "", Cerr{"context.getPkg", err}
	}

	imports, err := pkg.GetImports()
	if err != nil {
		return "", Cerr{"pkg.GetImports", err}
	}

	importNames, err := c.installImports(imports)
	if err != nil {
		return "", Cerr{"installImports", err}
	}

	newName := pkg.Label()
	c.importRewrites[newName] = pkgName
	importNames[pkgName] = newName

	err = pkg.MockImports(importNames, c.cfg)
	if err != nil {
		return "", Cerr{"MockImports", err}
	}

	cfg := c.cfg.Mock(pkgName)

	err = MockInterfaces(c.tmpPath, pkgName, cfg)
	if err != nil {
		return "", Cerr{"MockInterfaces", err}
	}

	c.code = append(c.code, pkg.Loc())

	return newName, nil
}

func (c *Context) LinkPackagesFromFile(path string) error {
	pkgs, err := readPackages(path)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if err := c.LinkPackage(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) ExcludePackagesFromFile(path string) error {
	pkgs, err := readPackages(path)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		c.excludes[pkg] = true
	}

	return nil
}

func (c *Context) Run(command string, args ...string) error {
	// Install the packages inside the context

	if err := c.installPackages(); err != nil {
		return err
	}

	// Create a Command object

	cmd := c.insideCommand(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Wrap stdout and stderr with rewriters, to put the paths back to real
	// code, not our symlinks.

	if c.doRewrite {
		stdout := NewRewriter(os.Stdout)
		defer stdout.Close()
		stderr := NewRewriter(os.Stderr)
		defer stderr.Close()

		for _, loc := range c.code {
			stdout.Rewrite(loc.dst, loc.src)
			stderr.Rewrite(loc.dst, loc.src)
		}

		for marked, orig := range c.importRewrites {
			stdout.Rewrite(marked, orig)
			stderr.Rewrite(marked, orig)
		}

		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}

	// Then run the given command

	return cmd.Run()
}
