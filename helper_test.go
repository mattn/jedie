package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSTR(t *testing.T) {
	pongoSetup()
	tests := []struct {
		in  interface{}
		out string
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

func TestCopyFile(t *testing.T) {
	inFile, err := ioutil.TempFile(os.TempDir(), "dude")

	if err != nil {
		panic(err)
	}

	copyFile(inFile.Name(), "muhaha")

	if _, err := os.Stat("muhaha"); os.IsNotExist(err) {
		t.Errorf("expected %q actual does not exist with error %v", inFile.Name(), err)
		os.Remove(inFile.Name())
		return
	}

	os.Remove(inFile.Name())
	os.Remove("muhaha")
}

func TestCopyFileErr(t *testing.T) {
	in := ""

	_, err := copyFile(in, "")

	if err == nil {
		t.Errorf("expected copyfile to return nil")
	}

	inFile, err := ioutil.TempFile(os.TempDir(), "dude")

	if err != nil {
		panic(err)
	}

	defer os.Remove(inFile.Name())

	_, err = copyFile(inFile.Name(), "")

	if err == nil {
		t.Errorf("expected copyfile to return nil")
	}

}
