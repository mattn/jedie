package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		s   string
		in  string
		out bool
	}{
		{layoutDefault, "Jedie is Awesome!", true},
		{layoutPost, "layout: default", true},
		{cssSite, "body", true},
		{postsBlog, "layout: post", true},
		{topPage, "title: Your New Jedie Site", true},
		{rssXml, "<rss version", true},
		{configYml, "Your New Jedie Site", true},
	}

	for _, test := range tests {
		if strings.Contains(test.s, test.in) != test.out {
			t.Errorf("nuts %v", layoutDefault)
		}
	}
}

func TestGenerateScaffold(t *testing.T) {
	tempDir, err := ioutil.TempDir(".", "dude")

	if err != nil {
		panic(err)
	}

	generateScaffold(tempDir)

	os.RemoveAll(tempDir)
}
