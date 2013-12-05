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
			t.Errorf("expected %v actual %v", test.in, actual)
		}
	}
}

func TestJOIN(t *testing.T) {
	tests := []struct {
		left  string
		right string
		out string
	}{
		{"Dude", "Dude", "Dude/Dude"},
		{"/Dude", "/Dude", "/Dude/Dude"},
		{"/Dude", "Dude", "/Dude/Dude"},
	}

	for _, test := range tests {
		actual := join(test.left, test.right)
		if actual != test.out {
			t.Errorf("expected %s actual %s", test.out, actual)
		}
	}
}
