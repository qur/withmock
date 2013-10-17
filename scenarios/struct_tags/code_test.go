package code

import (
	"testing"
)

func TestTryMe(t *testing.T) {
	// Run the function we want to test
	s, found := TryMe("Wibble")

	if !found {
		t.Errorf("Expected field not found")
	}

	if s != `json:"wibble"` {
		t.Errorf("Returned string wrong: %s", s)
	}
}

func TestTryMe2(t *testing.T) {
	// Run the function we want to test
	s, found := TryMe("Bar")

	if !found {
		t.Errorf("Expected field not found")
	}

	if s != `bson:"bar"` {
		t.Errorf("Returned string wrong: %s", s)
	}
}
