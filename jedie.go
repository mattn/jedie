package main

import (
	"flag"
	"fmt"
	"github.com/flosch/pongo"
	"github.com/howeyc/fsnotify"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type config struct {
	baseurl     string `yaml:"baseurl"`
	source      string `yaml:"source"`
	destination string `yaml:"destination"`
	posts       string `yaml:"posts"`
	data        string `yaml:"data"`
	includes    string `yaml:"includes"`
	layouts     string `yaml:"layouts"`
	vars        pongo.Context
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
	cfg.baseurl = str(globalVariables["baseurl"])
	cfg.source = str(globalVariables["source"])
	cfg.destination = str(globalVariables["destination"])
	cfg.posts = str(globalVariables["posts"])
	cfg.data = str(globalVariables["data"])
	cfg.includes = str(globalVariables["includes"])
	cfg.layouts = str(globalVariables["layouts"])

	if cfg.source == "" {
		cfg.source = "."
	}
	if cfg.destination == "" {
		cfg.destination = "_site"
	}
	if cfg.posts == "" {
		cfg.posts = "_posts"
	}
	if cfg.data == "" {
		cfg.data = "_data"
	}
	if cfg.includes == "" {
		cfg.includes = "_includes"
	}
	if cfg.layouts == "" {
		cfg.layouts = "_layouts"
	}
	cfg.source, err = filepath.Abs(cfg.source)
	if err != nil {
		return err
	}
	cfg.destination, err = filepath.Abs(cfg.destination)
	if err != nil {
		return err
	}
	cfg.posts, err = filepath.Abs(cfg.posts)
	if err != nil {
		return err
	}
	cfg.data, err = filepath.Abs(cfg.data)
	if err != nil {
		return err
	}
	cfg.includes, err = filepath.Abs(cfg.includes)
	if err != nil {
		return err
	}
	cfg.layouts, err = filepath.Abs(cfg.layouts)
	if err != nil {
		return err
	}

	cfg.source = filepath.ToSlash(cfg.source)
	cfg.destination = filepath.ToSlash(cfg.destination)
	cfg.posts = filepath.ToSlash(cfg.posts)
	cfg.data = filepath.ToSlash(cfg.data)
	cfg.includes = filepath.ToSlash(cfg.includes)
	cfg.layouts = filepath.ToSlash(cfg.layouts)
	cfg.vars["site"] = pongo.Context{}
	return nil
}

func (cfg *config) toPageUrl(from string) string {
	return join(cfg.baseurl, filepath.ToSlash(from[len(cfg.source):]))
}

func (cfg *config) toDate(from string) time.Time {
	fi, err := os.Stat(from)
	if err != nil {
		return time.Now()
	}
	name := filepath.Base(from)
	if len(name) <= 11 {
		return fi.ModTime()
	}
	date, err := time.Parse("2006-01-02-", name[:11])
	if err != nil {
		return fi.ModTime()
	}
	return date
}

func (cfg *config) toPostUrl(from string) string {
	ext := filepath.Ext(from)
	name := filepath.Base(from)
	name = name[0:len(name)-len(ext)] + ".html"
	if len(name) > 11 {
		date, err := time.Parse("2006-01-02-", name[:11])
		if err == nil {
			return join(cfg.baseurl, date.Format("/2006/01/02/")+name[11:])
		}
	}
	return join(cfg.baseurl, name)
}

func (cfg *config) toPage(from string) string {
	return filepath.ToSlash(filepath.Join(cfg.destination, from[len(cfg.source):]))
}

func (cfg *config) toPost(from string) string {
	ext := filepath.Ext(from)
	name := filepath.Base(from)
	name = name[0:len(name)-len(ext)] + ".html"
	if len(name) > 11 {
		date, err := time.Parse("2006-01-02-", name[:11])
		if err == nil {
			return filepath.ToSlash(filepath.Join(cfg.destination, date.Format("/2006/01/02/")+name[11:]))
		}
	}
	return filepath.ToSlash(filepath.Join(cfg.destination, name))
}

func (cfg *config) convertFile(src, dst string) error {
	dir := filepath.Dir(dst)
	_, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}
	}
	ext := filepath.Ext(src)
	if isConvertable(src) {
		if isMarkdown(src) {
			dst = dst[0:len(dst)-len(filepath.Ext(dst))] + ".html"
		}

		vars := pongo.Context{"content": ""}
		for {
			for k, v := range cfg.vars {
				vars[k] = v
			}
			pageVars := map[string]interface{}{}
			content, err := parseFile(src, pageVars)
			if err != nil {
				return err
			}
			for k, v := range pageVars {
				vars[k] = v
			}
			vars["post"] = map[string]interface{}{
				"date":  cfg.toDate(src),
				"url":   cfg.toPostUrl(src),
				"title": str(vars["title"]),
			}
			vars["page"] = map[string]interface{}{
				"date":  cfg.toDate(src),
				"url":   cfg.toPostUrl(src),
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

			if isMarkdown(src) {
				renderer := blackfriday.HtmlRenderer(0, "", "")
				vars["content"] = string(blackfriday.Markdown([]byte(content), renderer, extensions))
			} else {
				vars["content"] = content
			}
			if str(vars["layout"]) == "" || str(vars["layout"]) == "nil" {
				break
			}
			src = filepath.ToSlash(filepath.Join(cfg.layouts, str(vars["layout"])+".html"))
			ext = filepath.Ext(src)
			content = str(vars["content"])
			vars["content"] = content
			vars["layout"] = ""
		}

		err = ioutil.WriteFile(dst, []byte(str(vars["content"])), 0644)
	} else {
		switch ext {
		case ".yml", ".go", ".exe":
			return nil
		}
		_, err = copyFile(src, dst)
	}
	return err
}

func (cfg *config) New(p string) error {
	return generateScaffold(p)
}

func (cfg *config) Build() error {
	pongoSetup()

	var err error
	var pageFiles []string
	pages := []map[string]interface{}{}
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
		} else {
			if dot != '.' && dot != '_' {
				pageFiles = append(pageFiles, from)
				vars := map[string]interface{}{}
				vars["url"] = cfg.toPageUrl(from)
				vars["date"] = info.ModTime()
				pages = append(pages, vars)
			}
		}
		return err
	})
	checkFatal(err)

	categories := pongo.Context{}
	var postFiles []string
	posts := []pongo.Context{}
	err = filepath.Walk(cfg.posts, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.posts {
			return err
		}
		if info.IsDir() {
			return err
		}
		from := filepath.ToSlash(name)
		if !isConvertable(from) {
			return err
		}
		vars := pongo.Context{}
		_, err = parseFile(from, vars)
		if err != nil {
			return err
		}
		fi, err := os.Stat(from)
		if err != nil {
			return err
		}
		postFiles = append(postFiles, from)
		vars["url"] = cfg.toPostUrl(from)
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
		return err
	})
	checkFatal(err)

	cfg.vars["site"].(pongo.Context)["baseurl"] = cfg.baseurl
	cfg.vars["site"].(pongo.Context)["time"] = time.Now()
	cfg.vars["site"].(pongo.Context)["pages"] = pages
	cfg.vars["site"].(pongo.Context)["posts"] = posts
	cfg.vars["site"].(pongo.Context)["categories"] = categories
	cfg.vars["site"].(pongo.Context)["data"] = pongo.Context{}

	fis, err := ioutil.ReadDir(cfg.data)
	if err == nil {
		for _, fi := range fis {
			ext := filepath.Ext(fi.Name())
			var data interface{}
			switch ext {
			case ".yaml", ".yml":
				b, err := ioutil.ReadFile(filepath.Join(cfg.data, fi.Name()))
				if err != nil {
					return err
				}
				err = goyaml.Unmarshal(b, &data)
				if err == nil {
					name := fi.Name()
					name = name[0:len(name)-len(ext)]
					cfg.vars["site"].(pongo.Context)["data"].(pongo.Context)[name] = data
				}
			}
		}
	}

	if _, err := os.Stat(cfg.destination); err != nil {
		err = os.MkdirAll(cfg.destination, 0755)
		checkFatal(err)
	}

	for _, from := range pageFiles {
		to := cfg.toPage(from)
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		checkFatal(err)
	}

	for _, from := range postFiles {
		to := cfg.toPost(from)
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		checkFatal(err)
	}
	return nil
}

func (cfg *config) Serve() error {
	err := cfg.Build()
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = filepath.Walk(cfg.source, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.destination {
			return err
		}
		if info.IsDir() {
			if filepath.HasPrefix(name, ".") || filepath.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			err = watcher.WatchFlags(name, fsnotify.FSN_ALL)
			if err != nil {
				return err
			}
		}
		return nil
	})
	checkFatal(err)
	go func() {
		fired := false
		for {
			select {
			case e := <-watcher.Event:
				from := filepath.ToSlash(e.Name)
				if filepath.HasPrefix(from, cfg.destination) {
					continue
				}
				to := ""

				if filepath.HasPrefix(from, cfg.posts) {
					to = cfg.toPost(from)
				} else if filepath.HasPrefix(from, cfg.source) {
					to = cfg.toPage(from)
				}
				if to != "" {
					if !fired {
						fired = true
						go func(from, to string) {
							fired = false
							select {
							case <-time.After(100 * time.Millisecond):
								fired = false
								fmt.Println(from, "=>", to)
								cfg.convertFile(from, to)
							}
						}(from, to)
					}
				}
			case err := <-watcher.Error:
				log.Println("Error:", err)
			}
		}
	}()
	return http.ListenAndServe(":4000", http.FileServer(http.Dir(cfg.destination)))
}

func checkFatal(err error) {
	if err != nil {
		log.Fatal(err)
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
		checkFatal(err)
	case flag.Arg(0) == "build":
		err = cfg.load("_config.yml")
		checkFatal(err)
		err = cfg.Build()
		checkFatal(err)
	case flag.Arg(0) == "serve":
		err = cfg.load("_config.yml")
		checkFatal(err)
		err = cfg.Serve()
		checkFatal(err)
	default:
		flag.Usage()
	}
}
