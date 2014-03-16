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

	tmpRoot string

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
	packages map[string]*Package
}

type codeLoc struct {
	src, dst string
}

func getTmpPath(tmpDir string) string {
	return filepath.Join(tmpDir, "path")
}

func getTmpRoot(tmpDir string) string {
	return filepath.Join(tmpDir, "root")
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
		tmpRoot:        getTmpRoot(tmpDir),
		tmpDir:         tmpDir,
		stdlibImports:  stdlibImports,
		removeTmp:      true,
		processed:      make(map[string]bool),
		importRewrites: make(map[string]string),
		doRewrite:      true,
		cfg:            &Config{},
		cache:          cache,
		packages:       make(map[string]*Package),
		// create excludes already including gomock, as we can't mock it.
		excludes: map[string]bool{
			"github.com/qur/gomock/gomock": true,
			"github.com/qur/gomock/interfaces": true,
		},
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
		if strings.HasPrefix(env[i], "GOROOT=") {
			env[i] = "__IGNORE="
		}
	}

	// Setup the environment variables that we want
	env = append(env, "GOPATH=" + c.tmpPath)
	env = append(env, "GOROOT=" + c.tmpRoot)
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

func (c *Context) mockStdlib() error {
	list, err := GetOutput("go", "list", "std")
	if err != nil {
		return Cerr{"GetOutput(\"go list std\")", err}
	}

	pkgs := make(map[string]*Package)
	deps := make(map[string]map[string]bool)

	runtimerw := NewRewriter(nil)

	// We want to intercept goroutine creation
	runtimerw.Rewrite("runtime·newproc1(FuncVal ", "_real_newproc1(FuncVal ")

	for _, line := range strings.Split(list, "\n") {
		pkgName := strings.TrimSpace(line)
		label := markImport(pkgName, normalMark)

		if strings.HasPrefix(pkgName, "cmd/") {
			// Ignore commands
			continue
		}

		pkg, err := NewStdlibPackage(pkgName, label, c.tmpDir, c.goRoot, runtimerw)
		if err != nil {
			return Cerr{"NewPackage", err}
		}

		pkgs[pkgName] = pkg
		deps[pkgName] = make(map[string]bool)
	}

	for pkgName, pkg := range pkgs {
		imports, err := GetOutput("go", "list", "-f", "{{range .Deps}}{{println .}}{{end}}", pkgName)
		if err != nil {
			return Cerr{"GetOuput(go list .Deps)", err}
		}

		for _, name := range strings.Split(imports, "\n") {
			name = strings.TrimSpace(name)

			if name == "" || name == "github.com/qur/gomock/interfaces" {
				continue
			}
			_, found := deps[name]
			if !found {
				return fmt.Errorf("missing dependency %s for %s", name, pkgName)
			}
			deps[pkgName][name] = true
		}

		if  pkgName == "runtime" || pkgName == "unsafe" || strings.HasPrefix(pkgName, "runtime/") {
			// We need special handling for the unsafe and runtime packages.
			// All packages (apart from unsafe and runtime) get an automatic
			// dependancy on runtime, which itself depends on unsafe.  This
			// means we can't mock either of these packages.  In addition the
			// runtime package actually injects functions into other packages in
			// the stdlib - so we need to know what they are so that we can
			// rename them when we setup our copy of runtime.
			continue
		}

		if pkgName == "testing" || strings.HasPrefix(pkgName, "testing/") {
			// We don't want to mock testing - that just doesn't make sense ...
			if err := pkg.DisableAllMocks();  err != nil {
				return Cerr{"pkg.Link", err}
			}
		} else {
			cfg := c.cfg.Mock(pkgName)
			if _, err := pkg.Gen(false, cfg); err != nil {
				return Cerr{"pkg.Gen", err}
			}
		}
	}

	// Now that we have done all the other packages we can do the runtime and
	// unsafe packages.
	for _, pkgName := range []string{"unsafe", "runtime"} {
		pkg := pkgs[pkgName]

		_, err = pkg.Rewrite()
		if err != nil {
			return Cerr{"pkg.Rewrite", err}
		}
	}

	// Add some code to enable/disable mocking to the runtime package
	loc := pkgs["runtime"].Loc()
	if err := addMockController(loc.dst); err != nil {
		return Cerr{"addMockController", err}
	}

	// Before we can install the packages we need to get the toolchain
	toolSrc := filepath.Join(c.goRoot, "pkg", "tool")
	toolDst := filepath.Join(c.tmpRoot, "pkg", "tool")
	symlinkPackage(toolSrc, toolDst)

	// Apparently runtime/cgo needs cmd/cgo ...
	cgoSrc := filepath.Join(c.goRoot, "src", "cmd", "cgo")
	cgoDst := filepath.Join(c.tmpRoot, "src", "cmd", "cgo")
	symlinkPackage(cgoSrc, cgoDst)

	// We also need some apparently random extra stuff
	for _, path := range []string{
		"pkg/linux_amd64/runtime.h",
		"pkg/linux_amd64/cgocall.h",
		"src/cmd/ld/textflag.h",
	} {
		src := filepath.Join(c.goRoot, path)
		dst := filepath.Join(c.tmpRoot, path)

		dstDir := filepath.Dir(dst)

		if err := os.MkdirAll(dstDir, 0700); err != nil {
			return Cerr{"os.MkDirAll", err}
		}

		if err := os.Symlink(src, dst); err != nil {
			return Cerr{"os.Symlink", err}
		}
	}

	// Install the packages in reverse depedency order
	last := len(deps)
	for len(deps) > 0 {
		inst := []string{}
		for name, needs := range deps {
			if len(needs) > 0 {
				continue
			}

			cmd := c.insideCommand("go", "install", name)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
					name, err, out)
			}

			inst = append(inst, name)
		}

		// remove installed packages from deps
		for _, name := range inst {
			delete(deps, name)
		}

		// remove installed packages from any needs
		for _, needs := range deps {
			for _, name := range inst {
				delete(needs, name)
			}
		}

		if len(deps) == last {
			return fmt.Errorf("Unable to resolve dependencies for stdlib")
		}

		last = len(deps)
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

	for name := range imports {
		label := markImport(name, normalMark)
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

			if c.stdlibImports[name] {
				// Ignore stdlib packages, we deal with them separately.
				continue
			}

			pkg, err := c.getPkg(name, label)
			if err != nil {
				return nil, Cerr{"context.getPkg", err}
			}

			cfg := c.cfg.Mock(name)

			if c.excludes[name] {
				// this package has been specifically excluded from mocking, so
				// we just link it, even if mocked is indicated.
				if _, err := pkg.Link(); err != nil {
					return nil, Cerr{"pkg.Link", err}
				}
				continue
			}

			if imports[name].IsReplace() {
				// Install the requested package in place of the
				// package that the code thinks it wants.
				srcPath := imports[name].path
				pkgImports, err := pkg.Replace(srcPath)
				if err != nil {
					return nil, Cerr{"pkg.Replace", err}
				}

				// Update imports from the package we just processed, but it
				// can only add actual packages, not mocks
				c.wantToProcess(false, pkgImports)

				continue
			}

			// Process the package and get it's imports
			pkgImports, err := pkg.Gen(mock, cfg)
			if err != nil {
				return nil, Cerr{"pkg.Gen", err}
			}

			// Update imports from the package we just processed, but it can
			// only add actual packages, not mocks
			c.wantToProcess(false, pkgImports)
		}
	}

	return names, nil
}

func (c *Context) getPkg(pkgName, label string) (*Package, error) {
	pkg, found := c.packages[label]
	if found {
		return pkg, nil
	}

	pkg, err := NewPackage(pkgName, label, c.tmpDir, c.goPath)
	if err != nil {
		return nil, Cerr{"NewPackage", err}
	}

	c.packages[label] = pkg

	return pkg, nil
}

func (c *Context) LinkPackage(pkgName string) error {
	pkg, err := c.getPkg(pkgName, markImport(pkgName, normalMark))
	if err != nil {
		return Cerr{"c.getPkg", err}
	}

	_, err = pkg.Link()
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

func (c *Context) addRequiredPackage(name string) error {
	label := markImport(name, normalMark)

	if _, found := c.packages[label]; found {
		return nil
	}

	pkg, err := c.getPkg(name, label)
	if err != nil {
		return Cerr{"context.getPkg", err}
	}

	if _, err := pkg.Link(); err != nil {
		return Cerr{"pkg.Link", err}
	}

	return nil
}

func (c *Context) addRequiredPackages() error {
	for _, name := range []string{
		"github.com/qur/gomock/interfaces",
	} {
		if err := c.addRequiredPackage(name); err != nil {
			return Cerr{"c.addRequiredPackage", err}
		}
	}

	return nil
}

func (c *Context) Run(command string, args ...string) error {
	// Make sure required packages are installed

	if err := c.addRequiredPackages(); err != nil {
		return Cerr{"c.addRequiredPackages", err}
	}

	// Create a mocked version of the stdlib

	if err := c.mockStdlib(); err != nil {
		return Cerr{"c.mockStdlib", err}
	}

	// Install the packages inside the context

	if err := c.installPackages(); err != nil {
		return Cerr{"c.installPackages", err}
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
