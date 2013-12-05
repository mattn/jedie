package main

import (
	"testing"
)

func TestSTR(t *testing.T) {
	tests := []struct {
		in  interface{}
		out interface{}
	}{
		{"", ""},
		{1, ""},
		{"Dude", "Dude"},
	}

	for _, test := range tests {
		actual := str(test.in)
		if actual != test.out {
			t.Errorf("expected %v actual %v", test.in, test.out)
		}
	}
}
