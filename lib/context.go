package lib

import (
	"flag"
	"io/ioutil"
	"fmt"
	"os"
	"path/filepath"
	"os/exec"
)

var (
	work = flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting")
)

type Context struct {
	goPath string
	goRoot string

	tmpPath string
	origPath string

	tmpDir string
	removeTmp bool

	rootImports map[string]bool
	imports []string

	needsInstall []string
	standardMocks map[string]string

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

	rootImports, err := GetRootImports(goRoot)
	if err != nil {
		return nil, err
	}

	// Now we need to sort out some temporary directories to work with

	tmpDir, err := ioutil.TempDir("", "withmock")
	if err != nil {
		return nil, err
	}
	if *work {
		fmt.Fprintf(os.Stderr, "WORK=%s\n", tmpDir)
	}

	// Build and return the context

	return &Context{
		goPath: goPath,
		goRoot: goRoot,
		origPath: os.Getenv("GOPATH"),
		tmpPath: filepath.Join(tmpDir, "path"),
		tmpDir: tmpDir,
		rootImports: rootImports,
		removeTmp: !*work,
		needsInstall: []string{},
		standardMocks: make(map[string]string),
	}, nil
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

	return nil
}

func (c *Context) installPackages() error {
	for _, name := range c.needsInstall {
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

	// TODO: How do we know where to go?
	if err := os.Chdir(c.code[0].dst); err != nil {
		return err
	}

	return nil
}

func (c *Context) installImports(path string) error {
	imports, err := GetImports(path, true)
	if err != nil {
		return err
	}

	// Now we create a new GOPATH that contains all the packages that are
	// imported by the code we are interested in (generating mocks if
	// appropriate)

	processed := make(map[string]bool)
	complete := false

	for !complete {
		complete = true
		for name, mock := range imports {
			if processed[name] {
				continue
			}
			complete = false

			processed[name] = true

			if c.rootImports[name] {
				// Ignore standard packages unless mocked
				if mock {
					mockName, err := MockStandard(c.goRoot, c.tmpPath, name)
					if err != nil {
						return err
					}
					c.standardMocks[name] = mockName
					c.needsInstall = append(c.needsInstall, mockName)
				}
				continue
			}

			c.needsInstall = append(c.needsInstall, name)

			mkpkg := LinkPkg
			if mock {
				mkpkg = GenMockPkg
			}

			// Process the package and get it's imports
			pkgImports, err := mkpkg(c.goPath, c.tmpPath, name)
			if err != nil {
				return err
			}

			// Update imports from the package we just processed
			for name, mock := range pkgImports {
				imports[name] = imports[name] || mock
			}
		}
	}

	return nil
}

func (c *Context) LinkPkg(pkg string) error {
	_, err := LinkPkg(c.goPath, c.tmpPath, pkg)
	return err
}

func (c *Context) AddPackage(path string) error {
	if err := c.installImports(path); err != nil {
		return err
	}

	pkgName, err := GetOutput("go", "list", path)
	if err != nil {
		return err
	}

	codeDest := filepath.Join(c.tmpPath, "src", pkgName)
	codeSrc, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	err = MockImports(codeSrc, codeDest, c.standardMocks)
	if err != nil {
		return err
	}

	c.code = append(c.code, codeLoc{codeSrc, codeDest})

	return nil
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

	// Then run the given command

	cmd := exec.Command(command, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
