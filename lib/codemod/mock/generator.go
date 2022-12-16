package mock

import (
	"bytes"
	"context"
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/imports"

	"github.com/qur/withmock/lib/proxy/api"
)

type MockGenerator struct {
	prefix  string
	scratch string
	s       api.Store
}

func NewMockGenerator(prefix, scratch string, s api.Store) *MockGenerator {
	return &MockGenerator{
		prefix:  prefix,
		scratch: scratch,
		s:       s,
	}
}

func (m *MockGenerator) GenSource(ctx context.Context, mod, ver, _, _, dest string) error {
	origMod, err := m.stripPrefix(mod)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()

	mi, err := m.getModInfo(ctx, fset, origMod, ver)
	if err != nil {
		return fmt.Errorf("failed to get mod info (%s, %s): %w", origMod, ver, err)
	}

	interfaces, err := mi.resolveAllInterfaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve all interfaces: %w", err)
	}

	if interfaces == 0 {
		log.Printf("Mock - no interfaces found: %s@v%s", mod, ver)
		return api.UnknownVersion(mod, ver)
	}

	if err := mi.renderMocks(ctx, dest); err != nil {
		return fmt.Errorf("failed to render mocks: %w", err)
	}

	if err := mi.writeModFile(ctx, dest, mod); err != nil {
		return fmt.Errorf("failed to write mod file: %w", err)
	}

	return nil
}

func (m *MockGenerator) stripPrefix(mod string) (string, error) {
	if !strings.HasPrefix(mod, m.prefix) {
		return "", fmt.Errorf("module '%s' didn't have prefix '%s'", mod, m.prefix)
	}
	return mod[len(m.prefix):], nil
}

func save(dest string, fset *token.FileSet, node *dst.File) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := &bytes.Buffer{}
	if err := decorator.Fprint(buf, node); err != nil {
		return err
	}
	formatted, err := imports.Process(dest, buf.Bytes(), nil)
	if err != nil {
		return err
	}
	if _, err := f.Write(formatted); err != nil {
		return err
	}
	return nil
}
