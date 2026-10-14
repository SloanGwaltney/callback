package main

import (
	"errors"
	"testing"
)

func TestNoOpenerOpensNothing(t *testing.T) {
	if err := (noOpener{}).Open("https://example.com"); !errors.Is(err, errNoOpening) {
		t.Errorf("err = %v, want errNoOpening", err)
	}
}
