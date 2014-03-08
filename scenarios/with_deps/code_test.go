package withdeps

import (
	"go/build"
	"path/filepath"
	"testing"

	"fmt" // mock

	"github.com/qur/gomock/gomock"
)

func testDataPath() string {
	pkg, err := build.Import("github.com/qur/withmock/scenarios/with_deps/deps", "", build.FindOnly)
	if err != nil {
		panic(err)
	}
	return pkg.Dir
}

func TestShow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fmt.MOCK().SetController(ctrl)

	// We need to find out test data file
	path := filepath.Join(testDataPath(), "count")

	// The count file contains three lines that we expect to be printed.
	fmt.EXPECT().Printf("%d: %s\n", 1, "one")
	fmt.EXPECT().Printf("%d: %s\n", 2, "two")
	fmt.EXPECT().Printf("%d: %s\n", 3, "three")

	// Run the function we want to test
	err := Show(path)

	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
}
