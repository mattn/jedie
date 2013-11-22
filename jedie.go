package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/flosch/pongo"
	"github.com/russross/blackfriday"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
	"net/url"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS

type config struct {
	baseUrl     string `yaml:"baseUrl"`
	source      string `yaml:"source"`
	destination string `yaml:"destination"`
	vars        pongo.Context
}

func str(s interface{}) string {
	if ss, ok := s.(string); ok {
		return ss
	}
	return ""
}

func (cfg *config) load(file string) error {
	b, err := ioutil.ReadFile("_config.yml")
	if err != nil {
		return err
	}

	var globalVariables pongo.Context
	err = goyaml.Unmarshal(b, &globalVariables)
	if err != nil {
		return err
	}

	cfg.vars = globalVariables
	cfg.baseUrl = str(globalVariables["baseUrl"])
	cfg.source = str(globalVariables["source"])
	cfg.destination = str(globalVariables["destination"])

	if cfg.source == "" {
		cfg.source = ""
	}
	if cfg.destination == "" {
		cfg.destination = "_site"
	}
	cfg.source, err = filepath.Abs(cfg.source)
	if err != nil {
		return err
	}
	cfg.destination, err = filepath.Abs(cfg.destination)
	if err != nil {
		return err
	}

	cfg.source = filepath.ToSlash(cfg.source)
	cfg.destination = filepath.ToSlash(cfg.destination)
	cfg.vars["site"] = pongo.Context{}
	return nil
}

func (cfg *config) toUrl(from string) string {
	return cfg.baseUrl + filepath.ToSlash(from[len(cfg.source):])
}

func (cfg *config) toPage(from string) string {
	return filepath.ToSlash(filepath.Join(cfg.destination, from[len(cfg.source):]))
}

func (cfg *config) toPost(from string) string {
	base := filepath.ToSlash(filepath.Join(cfg.source, "_posts"))
	// TODO Separate permalink as %Y/%m/%d/title.html
	return filepath.ToSlash(filepath.Join(cfg.destination, from[len(base):]))
}

func (cfg *config) convertFile(src, dst string) error {
	var err error
	ext := filepath.Ext(src)
	switch ext {
	case ".yml", ".go", ".exe":
		return nil
	case ".html", ".md", ".mkd":
		dst = dst[0:len(dst)-len(filepath.Ext(dst))] + ".html"
		fi, err := os.Stat(src)
		if err != nil {
			return err
		}

		vars := pongo.Context{"content": ""}
		for {
			for k, v := range cfg.vars {
				vars[k] = v
			}
			pageVars := pongo.Context{}
			content, err := parseFile(src, pageVars)
			if err != nil {
				return err
			}
			for k, v := range pageVars {
				vars[k] = v
			}
			vars["post"] = pongo.Context{
				"date": fi.ModTime(),
				"url": cfg.toUrl(src),
				"title": str(vars["title"]),
			}
			vars["page"] = pongo.Context{
				"date": fi.ModTime(),
				"url": cfg.toUrl(src),
				"title": str(vars["title"]),
			}
			if content != "" {
				ps := new(string)
				*ps = content
				// TODO The variables must be hidden for the each posts/pages?
				//old := cfg.vars
				//cfg.vars = vars
				tpl, err := pongo.FromString(str(vars["layout"]), ps, include(cfg, vars))
				//cfg.vars = old
				if err == nil {
					output, err := tpl.Execute(&vars)
					if err == nil && output != nil {
						content = *output
					} else {
						return err
					}
				} else {
					return err
				}
			}

			if ext == ".md" || ext == ".mkd" {
				renderer := blackfriday.HtmlRenderer(0, "", "")
				vars["content"] = string(blackfriday.Markdown([]byte(content), renderer, extensions))
			} else {
				vars["content"] = content
			}
			if str(vars["layout"]) == "" {
				break
			}
			src = filepath.ToSlash(filepath.Join(cfg.source, "_layouts", str(vars["layout"])+".html"))
			ext = filepath.Ext(src)
			content = str(vars["content"])
			vars["content"] = content
			vars["layout"] = ""
		}

		return ioutil.WriteFile(dst, []byte(str(vars["content"])), 0644)
	}
	_, err = copyFile(src, dst)
	return err
}

func (cfg *config) createPost(src, dst string) error {
	var err error
	ext := filepath.Ext(src)
	switch ext {
	case ".yml", ".go", ".exe":
		return nil
	case ".html", ".md", ".mkd":
		return cfg.convertFile(src, dst)
	}
	_, err = copyFile(src, dst)
	return err
}

func (cfg *config) New(p string) error {
	content := "name: Your New Jedie Site\n"
	err := ioutil.WriteFile(filepath.Join(p, "_config.yml"), []byte(content), 0644)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(p, "_layouts"), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(p, "_layouts", "default.html"), []byte(layoutDefault), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(p, "_layouts", "post.html"), []byte(layoutPost), 0644)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(p, "css"), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(p, "css", "site.css"), []byte(cssSite), 0644)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(p, "_posts"), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(p, "_posts", time.Now().Format("2006-01-02-welcome-to-jedie") + ".md"), []byte(postsBlog), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(p, "index.html"), []byte(topPage), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (cfg *config) Build() error {
	pongoSetup()

	var err error
	var pageFiles []string
	pages := []pongo.Context{}
	err = filepath.Walk(cfg.source, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.source {
			return err
		}

		from := filepath.ToSlash(name)
		dot := filepath.Base(name)[0]
		if info.IsDir() {
			if from == cfg.destination || dot == '.' || dot == '_' {
				return filepath.SkipDir
			}
			err = os.MkdirAll(cfg.toPage(from), 0755)
		} else {
			if dot != '.' && dot != '_' {
				pageFiles = append(pageFiles, from)
				vars := pongo.Context{}
				vars["url"] = cfg.toUrl(from)
				vars["date"] = info.ModTime()
				pages = append(pages, vars)
			}
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	categories := pongo.Context{}
	var postFiles []string
	posts := []pongo.Context{}
	base := filepath.ToSlash(filepath.Join(cfg.source, "_posts"))
	err = filepath.Walk(base, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == base {
			return err
		}

		if !info.IsDir() {
			from := filepath.ToSlash(name)
			vars := pongo.Context{}
			ext := filepath.Ext(from)
			switch ext {
			case ".html", ".md", ".mkd":
				_, err = parseFile(from, vars)
				if err != nil {
					return err
				}
				fi, err := os.Stat(from)
				if err != nil {
					return err
				}
				postFiles = append(postFiles, from)
				from = from[0:len(from)-len(ext)] + ".html"
				vars["url"] = cfg.toUrl(filepath.Join(cfg.source, from[len(base):]))
				vars["date"] = fi.ModTime()
				if category, ok := vars["category"]; ok {
					cname := str(category)
					categorizedPosts := categories[cname]
					if categorizedPosts == nil {
						categorizedPosts = []pongo.Context{}
					}
					categorizedPosts = append(categorizedPosts.([]pongo.Context), vars)
					categories[cname] = categorizedPosts
				} else {
					posts = append(posts, vars)
				}
			}
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	cfg.vars["site"].(pongo.Context)["pages"] = pages
	cfg.vars["site"].(pongo.Context)["posts"] = posts
	cfg.vars["site"].(pongo.Context)["categories"] = categories

	if _, err := os.Stat(cfg.destination); err != nil {
		err = os.MkdirAll(cfg.destination, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, from := range pageFiles {
		to := cfg.toPage(from)
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, from := range postFiles {
		ext := filepath.Ext(from)
		to := filepath.ToSlash(filepath.Join(cfg.destination, from[len(base):]))
		to = to[0:len(to)-len(ext)] + ".html"
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func (cfg *config) Serve() error {
	err := cfg.Build()
	if err != nil {
		return err
	}
	return http.ListenAndServe(":4000", http.FileServer(http.Dir(cfg.destination)))
}

func parseFile(file string, vars pongo.Context) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	content := string(b)
	lines := strings.Split(content, "\n")
	if len(lines) > 2 && lines[0] == "---" {
		var line string
		var n int
		var yaml string
		for n, line = range lines[1:] {
			if line == "---" {
				break
			}
			yaml += line + "\n"
		}
		err = goyaml.Unmarshal(b, &vars)
		if err != nil {
			return "", err
		}
		content = strings.Join(lines[n+2:], "\n")
	}
	return content, nil
}

func include(cfg *config, vars pongo.Context) func(*string) (*string, error) {
	return func(loc *string) (*string, error) {
		inc := filepath.ToSlash(filepath.Join(cfg.source, "_includes", *loc))
		tpl, err := pongo.FromFile(inc, include(cfg, vars))
		if err != nil {
			return nil, err
		}
		return tpl.Execute(&vars)
	}
}

func copyFile(src, dst string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer df.Close()
	return io.Copy(df, sf)
}

func pongoSetup() {
	pongo.Filters["safe"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		return value, nil
	}
	pongo.Filters["escape"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		str, is_str := value.(string)
		if !is_str {
			return nil, errors.New(fmt.Sprintf("%v (%T) is not of type string", value, value))
		}
		return url.QueryEscape(str), nil
	}
	pongo.Filters["date_to_string"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		date, ok := value.(time.Time)
		if !ok {
			return nil, errors.New(fmt.Sprintf("Date must be of type time.Time not %T ('%v')", value, value))
		}
		return date.Format("2006/01/02 03:04:05"), nil
	}
	pongo.Filters["date"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("Please provide a count of limit")
		}
		format, is_string := args[0].(string)
		if !is_string {
			return nil, errors.New(fmt.Sprintf("Format must be of type string, not %T ('%v')", args[0], args[0]))
		}

		var err error
		date, ok := value.(time.Time)
		if !ok {
			datestr, ok := value.(string)
			if !ok {
				return nil, errors.New(fmt.Sprintf("Date must be of type time.Time or string, not %T ('%v')", value, value))
			}
			date, err = time.Parse(format, datestr)
			if err != nil {
				return nil, err
			}
		}
		format = strings.Replace(format, "%a", "Mon", -1)
		format = strings.Replace(format, "%A", "Monday", -1)
		format = strings.Replace(format, "%b", "Jan", -1)
		format = strings.Replace(format, "%B", "January", -1)
		format = strings.Replace(format, "%c", time.RFC3339, -1)
		format = strings.Replace(format, "%C", "06", -1)
		format = strings.Replace(format, "%d", "02", -1)
		format = strings.Replace(format, "%C", "01/02/06", -1)
		format = strings.Replace(format, "%e", "_1/_2/_6", -1)
		//format = strings.Replace(format, "%E", "", -1)
		format = strings.Replace(format, "%F", "06-01-02", -1)
		//format = strings.Replace(format, "%G", "", -1)
		//format = strings.Replace(format, "%g", "", -1)
		format = strings.Replace(format, "%h", "Jan", -1)
		format = strings.Replace(format, "%H", "15", -1)
		format = strings.Replace(format, "%I", "03", -1)
		//format = strings.Replace(format, "%j", "", -1)
		format = strings.Replace(format, "%k", "3", -1)
		format = strings.Replace(format, "%l", "_3", -1)
		format = strings.Replace(format, "%m", "01", -1)
		format = strings.Replace(format, "%M", "04", -1)
		format = strings.Replace(format, "%n", "\n", -1)
		//format = strings.Replace(format, "%O", "", -1)
		format = strings.Replace(format, "%p", "PM", -1)
		format = strings.Replace(format, "%P", "pm", -1)
		format = strings.Replace(format, "%r", "03:04:05 PM", -1)
		format = strings.Replace(format, "%R", "03:04", -1)
		//format = strings.Replace(format, "%s", "", -1)
		format = strings.Replace(format, "%S", "05", -1)
		format = strings.Replace(format, "%t", "\t", -1)
		format = strings.Replace(format, "%T", "15:04:05", -1)
		//format = strings.Replace(format, "%u", "", -1)
		//format = strings.Replace(format, "%U", "", -1)
		//format = strings.Replace(format, "%V", "", -1)
		//format = strings.Replace(format, "%W", "", -1)
		//format = strings.Replace(format, "%x", "", -1)
		//format = strings.Replace(format, "%X", "", -1)
		format = strings.Replace(format, "%y", "06", -1)
		format = strings.Replace(format, "%Y", "2006", -1)
		format = strings.Replace(format, "%z", "-0700", -1)
		format = strings.Replace(format, "%Z", "MST", -1)
		//format = strings.Replace(format, "%+", "", -1)
		format = strings.Replace(format, "%%", "%", -1)
		return date.Format(format), nil
	}
	pongo.Filters["limit"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		if len(args) != 1 {
			return nil, errors.New("Please provide a count of limit")
		}
		limit, is_int := args[0].(int)
		if !is_int {
			return nil, errors.New(fmt.Sprintf("Limit must be of type int, not %T ('%v')", args[0], args[0]))
		}

		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return rv.Slice(0, limit).Interface(), nil
		case reflect.String:
			return value, nil
		default:
			return nil, errors.New(fmt.Sprintf("Cannot join variable of type %T ('%v').", value, value))
		}
		panic("unreachable")
	}
}

func main() {
	flag.Usage = func() {
		fmt.Println(`
  NAME:

    jedie

  DESCRIPTION:

    Static site generator in golang

  COMMANDS:

    new                  Creates a new jedie site scaffold in PATH
    build                Build your site
    serve                Serve your site locally
`[1:])
	}
	flag.Parse()

	var cfg config
	var err error
	switch {
	case flag.Arg(0) == "new":
		p := flag.Arg(1)
		if p == "" {
			flag.Usage()
			os.Exit(1)
		}
		err = cfg.New(p)
		if err != nil {
			log.Fatal(err)
		}
	case flag.Arg(0) == "build":
		err = cfg.load("_config.yml")
		if err != nil {
			log.Fatal(err)
		}
		err = cfg.Build()
		if err != nil {
			log.Fatal(err)
		}
	case flag.Arg(0) == "serve":
		err = cfg.load("_config.yml")
		if err != nil {
			log.Fatal(err)
		}
		err = cfg.Serve()
		if err != nil {
			log.Fatal(err)
		}
	default:
		flag.Usage()
	}
}

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
