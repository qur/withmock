package lib

import (
	"io/ioutil"
	"fmt"
	"os"
	"path/filepath"
	"os/exec"
)

type Context struct {
	goPath string
	goRoot string

	tmpPath string
	origPath string

	tmpDir string
	removeTmp bool

	stdlibImports map[string]bool
	imports []string

	processed map[string]bool
	importRewrites map[string]string

	code []codeLoc
}

type codeLoc struct {
	src, dst string
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

	// Build and return the context

	return &Context{
		goPath: goPath,
		goRoot: goRoot,
		origPath: os.Getenv("GOPATH"),
		tmpPath: filepath.Join(tmpDir, "path"),
		tmpDir: tmpDir,
		stdlibImports: stdlibImports,
		removeTmp: true,
		processed: make(map[string]bool),
		importRewrites: make(map[string]string),
	}, nil
}

func (c *Context) KeepWork() {
	if c.removeTmp {
		fmt.Fprintf(os.Stderr, "WORK=%s\n", c.tmpDir)
		c.removeTmp = false
	}
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

func (c *Context) setGOPATH() error {
	if err := os.Setenv("GOPATH", c.tmpPath); err != nil {
		return err
	}
    if err := os.Setenv("REAL_GOPATH", c.origPath); err != nil {
		return err
	}

	return nil
}

func (c *Context) installPackages() error {
	for name := range c.processed {
		if c.stdlibImports[name] {
			// stdlib imports don't need installing
			continue
		}

		cmd := exec.Command("go", "install", name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to install '%s': %s\noutput:\n%s",
				name, err, out)
		}
	}

	return nil
}

func (c *Context) activate() error {
	if err := c.setGOPATH(); err != nil {
		return err
	}

	if err := c.installPackages(); err != nil {
		return err
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

func (c *Context) installImports(imports map[string]bool) (map[string]string, error) {
	// Now we create a new GOPATH that contains all the packages that are
	// imported by the code we are interested in (generating mocks if
	// appropriate)

	names := make(map[string]string)
	complete := false

	for !complete {
		complete = true
		for name, mock := range imports {
			label := markImport(name, normalMark)
			if mock {
				label = markImport(name, mockMark)
			}
			names[name] = label

			if c.processed[label] {
				continue
			}
			complete = false

			c.processed[label] = true

			if c.stdlibImports[name] {
				// Ignore standard packages unless mocked
				if mock {
					err := MockStandard(c.goRoot, c.tmpPath, name)
					if err != nil {
						return nil, err
					}
				}
				continue
			}

			mkpkg := LinkPkg
			if mock {
				mkpkg = GenMockPkg
			}

			// Process the package and get it's imports
			pkgImports, err := mkpkg(c.goPath, c.tmpPath, name)
			if err != nil {
				return nil, err
			}

			// Update imports from the package we just processed, but it can
			// only add actual packages, not mocks
			for name, _ := range pkgImports {
				imports[name] = imports[name] || false
			}
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

	return names, nil
}

func (c *Context) LinkPkg(pkg string) error {
	_, err := LinkPkg(c.goPath, c.tmpPath, pkg)
	return err
}

func (c *Context) AddPackage(pkgName string) (string, error) {
	path, err := GetOutput("go", "list", "-f", "{{.Dir}}", pkgName)
	if err != nil {
		return "", err
	}

	imports, err := GetImports(path, true)
	if err != nil {
		return "", err
	}

	importNames, err := c.installImports(imports)
	if err != nil {
		return "", err
	}

	newName := markImport(pkgName, testMark)
	c.importRewrites[newName] = pkgName

	codeDest := filepath.Join(c.tmpPath, "src", newName)
	codeSrc, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	err = MockImports(codeSrc, codeDest, importNames)
	if err != nil {
		return "", err
	}

	c.code = append(c.code, codeLoc{codeSrc, codeDest})

	return newName, nil
}

func (c *Context) Run(command string, args ...string) error {
	// Activate the context, which will update the environment to use our new
	// GOPATH, and cd into the appropriate path for the code of interest.

	if err := c.activate(); err != nil {
		return err
	}

	// Wrap stdout and stderr with rewriters, to put the paths back to real
	// code, not our symlinks.

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

	// Then run the given command

	cmd := exec.Command(command, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
