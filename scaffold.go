package main

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

You'll find this post in your `+"`_posts`"+` directory - edit this post and re-build (or run with the `+"`-w`"+` switch) to see your changes!
To add new posts, simply add a file in the `+"`_posts`"+` directory that follows the convention: YYYY-MM-DD-name-of-post.ext.

Jekyll also offers powerful support for code snippets:

`+"```"+`go
package main

import "fmt"

func main() {
	fmt.Println("こんにちわ世界")
}
`+"```"+`

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
