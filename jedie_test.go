package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTmpDir() string {
	dir, err := ioutil.TempDir(os.TempDir(), "jedie")
	if err != nil {
		panic(err)
	}
	return dir
}

func makeConfig(content string) string {
	dir := makeTmpDir()
	err := ioutil.WriteFile(filepath.Join(dir, "_config.yml"), []byte(strings.TrimSpace(content)), 0644)
	if err != nil {
		panic(err)
	}
	return dir
}

func TestLoad(t *testing.T) {
	dir := makeConfig(`
name: Your New Jedie Site
description: You love golang, I love golang
permalink: /foo/:categories/:month/:title.html
	`)
	defer os.RemoveAll(dir)

	cfg := config{}
	err := cfg.load(filepath.Join(dir, "_config.yml"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Name != "Your New Jedie Site" {
		t.Fatalf("Unexpected cfg.Name: %s", cfg.Name)
	}
}

func TestIsMarkdown(t *testing.T) {
	tests := []struct {
		in  string
		out bool
	}{
		{"dude.md", true},
		{"dude.mkd", true},
		{"dude.markdown", true},
		{"dude.dude", false},
	}

	cfg := config{}
	for _, test := range tests {
		actual := cfg.isMarkdown(test.in)
		if actual != test.out {
			t.Errorf("expected %v actual %v", test.out, actual)
		}
	}
}

func TestIsConvertable(t *testing.T) {
	tests := []struct {
		in  string
		out bool
	}{
		{"dude.html", true},
		{"dude.xml", true},
		{"dude.md", true},
		{"dude.markdown", true},
		{"dude.dude", false},
	}

	cfg := config{}
	for _, test := range tests {
		actual := cfg.isConvertable(test.in)
		if actual != test.out {
			t.Errorf("expected %v actual %v", test.out, actual)
		}
	}
}
