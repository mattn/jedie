package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var layoutDefault = `
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
	<link rel="stylesheet" href="/css/site.css" media="all">
    <title>Jedie is Awesome!</title>
</head>
<body>
{{ content }}
</body>
</html>
`[1:]

var layoutPost = `
---
layout: default
---
<h2>{{ page.title }}</h2>
<p class="meta">{{ page.date | date_to_string }}</p>

<div class="post">
{{ content }}
</div>
`[1:]

var cssSite = `
body {
	font-family: Sans;
}

h1 {
	color: darkgreen;
}
`[1:]

var postsBlog = (`
---
layout: post
title:  "Welcome to Jedie!"
date:   2013-11-22 21:42:47
---

You'll find this post in your ` + "`_posts`" + ` directory - edit this post and re-build (or run with the ` + "`-w`" + ` switch) to see your changes!
To add new posts, simply add a file in the ` + "`_posts`" + ` directory that follows the convention: YYYY-MM-DD-name-of-post.ext.

Jekyll also offers powerful support for code snippets:

    package main
    
    import "fmt"
    
    func main() {
	    fmt.Println("こんにちわ世界")
    }

Check out the [Jedie][jedie-gh] for more info.

[jedie-gh]: https://github.com/mattn/jedie
`)[1:]

var topPage = `
---
layout: default
title: Your New Jedie Site
---
<div id="home">
  <h1>Blog Posts</h1>
  <ul class="posts">
    {% for post in site.posts %}
      <li><span>{{ post.date | date_to_string }}</span> &raquo; <a href="{{ post.url }}">{{ post.title }}</a></li>
    {% endfor %}
  </ul>
</div>
`[1:]

var rssXml = `
---
layout: nil
---
<?xml version="1.0" encoding="utf-8" ?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>{{ site.name | xml_escape }}</title>
    <link>{{ site.description | xml_escape }}</link>
    <atom:link rel="self" type="application/rss+xml" href="{{ site.baseurl | xml_escape }}{{ page.url | xml_escape}}" />
    <description>{{ site.description | xml_escape }}</description>
    <pubDate>{{ site.time | date:"%a, %d %b %Y %H:%M:%S +0900" }}</pubDate>
    <lastBuildDate>{{ site.time | date:"%a, %d %b %Y %H:%M:%S +0900" }}</lastBuildDate>
    {% for post in site.posts | limit:25 %}
    <item>
      <title>{{ post.title | xml_escape }}</title>
      <link>{{ site.baseurl | xml_escape }}{{ post.url | xml_escape }}</link>
      <guid isPermaLink="false">tag:vim-jp.org,{{ post.date | date:"%Y/%m/%d" }}:{{ post.url | xml_escape }},rev:1</guid>
      <pubDate>{{ post.date | date:"%a, %d %b %Y %H:%M:%S +0900" }}</pubDate>
      <author>vim-jp</author>
      <description>{{ post.content | xml_escape }}</description>
    </item>
    {% endfor %}
  </channel> 
</rss>
`[1:]

var configYml = `
name: Your New Jedie Site
description: You love golang, I love golang
`[1:]

func generateScaffold(p string) error {
	directories := []string{"_layouts", "css", "_posts"}

	for _, directory := range directories {
		err := os.Mkdir(filepath.Join(p, directory), 0755)
		if err != nil {
			return err
		}
	}

	files := []struct {
		first       string
		last        string
		templateVar string
	}{
		{"_config.yml", "", configYml},
		{"_layouts", "default.html", layoutDefault},
		{"_layouts", "post.html", layoutPost},
		{"css", "site.css", cssSite},
		{"_posts", time.Now().Format("2006-01-02-welcome-to-jedie.md"), postsBlog},
		{"index.html", "", topPage},
		{"rss.xml", "", rssXml},
	}

	for _, file := range files {
		err := ioutil.WriteFile(filepath.Join(p, file.first, file.last), []byte(file.templateVar), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
