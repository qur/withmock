package lib

import (
	"os"
	"path/filepath"
)

var mockControlC = `#include "runtime.h"

void
·getG(uintptr ret) {
	ret = (uintptr)g;
	FLUSH(&ret);
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

// store disabled flags so that missing entries will be considered enabled.
var mockDisabled = map[uintptr]bool{}

func MockingDisabled() bool {
	return mockDisabled[getG()]
}

func copyMocking(src, dst uintptr) {
	if mockDisabled[src] {
		mockDisabled[dst] = true
	}
}

func RestoreMocking(val bool) {
	mockDisabled[getG()] = val
}

func EnableMocking() bool {
	id := getG()
	old := mockDisabled[id]
	delete(mockDisabled, id)
	return old
}

func DisableMocking() bool {
	id := getG()
	old := mockDisabled[id]
	mockDisabled[id] = true
	return old
}

`

func addMockController(dst string) error {
	cpath := filepath.Join(dst, "mock_control.c")
	cf, err := os.Create(cpath)
	if err != nil {
		return Cerr{"os.Create(cpath)", err}
	}
	defer cf.Close()

	_, err = cf.WriteString(mockControlC)
	if err != nil {
		return Cerr{"cf.WriteString", err}
	}

	gopath := filepath.Join(dst, "mock_control.go")
	gf, err := os.Create(gopath)
	if err != nil {
		return Cerr{"os.Create(gopath)", err}
	}
	defer gf.Close()

	_, err = gf.WriteString(mockControlGo)
	if err != nil {
		return Cerr{"gf.WriteString", err}
	}

	return nil
}
