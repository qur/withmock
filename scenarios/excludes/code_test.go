package code

import (
	"testing"
)

func TestShow(t *testing.T) {
	// Run the function we want to test
	ret := TryMe()

	if ret != "Not Mocked!" {
		t.Errorf("Unexpected return: %s", ret)
	}
}
