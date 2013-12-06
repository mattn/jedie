package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

	testFiles := []struct {
		in  string
		out bool
	}{
		{"_config.yml", true},
		{"_layouts/default.html", true},
		{"_layouts/post.html", true},
		{"css/site.css", true},
		{"_posts/" + time.Now().Format("2006-01-02-welcome-to-jedie.md"), true},
		{"index.html", true},
		{"rss.xml", true},
	}

	for _, test := range testFiles {
		if _, err := os.Stat(tempDir + "/" + test.in); os.IsNotExist(err) {
			t.Errorf("expected %s actual does not exist with error %v", test.in, err)
		}
	}

	os.RemoveAll(tempDir)
}
