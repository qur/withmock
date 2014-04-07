package lib

import (
	"os"
	"path/filepath"

	"github.com/qur/withmock/utils"
)

var mockControlC = `#include "runtime.h"

void
·getG(uintptr ret) {
	ret = (uintptr)g;
	FLUSH(&ret);
}

static Lock mockmu;

void
·lockMock(void) {
	runtime·lock(&mockmu);
}

void
·unlockMock(void) {
	runtime·unlock(&mockmu);
}

extern void ·copyMocking(uintptr, uintptr);
extern G* _real_newproc1(FuncVal *fn, byte *argp, int32 narg, int32 nret, void *callerpc);

G*
runtime·newproc1(FuncVal *fn, byte *argp, int32 narg, int32 nret, void *callerpc) {
	G *gp = _real_newproc1(fn, argp, narg, nret, callerpc);
	·copyMocking((uintptr)g, (uintptr)gp);
	return gp;
}
`

var mockControlGo = `package runtime

func getG() uintptr
func lockMock()
func unlockMock()

// store disabled flags so that missing entries will be considered enabled.
var mockDisabled = map[uintptr]bool{}

func MockingDisabled() bool {
	lockMock()
	defer unlockMock()

	return mockDisabled[getG()]
}

func copyMocking(src, dst uintptr) {
	lockMock()
	defer unlockMock()

	if mockDisabled[src] {
		mockDisabled[dst] = true
	}
}

func RestoreMocking(val bool) {
	lockMock()
	defer unlockMock()

	mockDisabled[getG()] = val
}

func EnableMocking() bool {
	id := getG()

	lockMock()
	defer unlockMock()

	old := mockDisabled[id]
	delete(mockDisabled, id)

	return old
}

func DisableMocking() bool {
	id := getG()

	lockMock()
	defer unlockMock()

	old := mockDisabled[id]
	mockDisabled[id] = true

	return old
}

`

func addMockController(dst string) error {
	cpath := filepath.Join(dst, "mock_control.c")
	cf, err := os.Create(cpath)
	if err != nil {
		return utils.Err{"os.Create(cpath)", err}
	}
	defer cf.Close()

	_, err = cf.WriteString(mockControlC)
	if err != nil {
		return utils.Err{"cf.WriteString", err}
	}

	gopath := filepath.Join(dst, "mock_control.go")
	gf, err := os.Create(gopath)
	if err != nil {
		return utils.Err{"os.Create(gopath)", err}
	}
	defer gf.Close()

	_, err = gf.WriteString(mockControlGo)
	if err != nil {
		return utils.Err{"gf.WriteString", err}
	}

	return nil
}
