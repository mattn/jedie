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
	"strings"
	"time"
)

type config struct {
	Baseurl     string `yaml:"baseurl"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Posts       string `yaml:"posts"`
	Data        string `yaml:"data"`
	Includes    string `yaml:"includes"`
	Layouts     string `yaml:"layouts"`
	Permalink   string `yaml:"permalink"`
	Host		string `yaml:"host"`
	Port		int `yaml:"port"`
	vars        pongo.Context
}

type page struct {
	path string
	vars map[string]interface{}
}

func (cfg *config) load(file string) error {
	b, err := ioutil.ReadFile("_config.yml")
	if err != nil {
		return err
	}

	err = goyaml.Unmarshal(b, cfg)
	if err != nil {
		return err
	}
	cfg.vars = pongo.Context{}

	if cfg.Source == "" {
		cfg.Source = "."
	}
	if cfg.Destination == "" {
		cfg.Destination = "_site"
	}
	if cfg.Posts == "" {
		cfg.Posts = "_posts"
	}
	if cfg.Data == "" {
		cfg.Data = "_data"
	}
	if cfg.Includes == "" {
		cfg.Includes = "_includes"
	}
	if cfg.Layouts == "" {
		cfg.Layouts = "_layouts"
	}
	if cfg.Port <= 0  {
		cfg.Port = 4000
	}
	if cfg.Permalink == "" {
		cfg.Permalink = "date"
	}
	switch cfg.Permalink {
	case "date":
		cfg.Permalink = "/:categories/:year/:month/:day/:title.html"
	case "pretty":
		cfg.Permalink = "/:categories/:year/:month/:day/:title/"
	case "none":
		cfg.Permalink = "/:categories/:title.html"
	}

	cfg.Source, err = filepath.Abs(cfg.Source)
	if err != nil {
		return err
	}
	cfg.Destination, err = filepath.Abs(cfg.Destination)
	if err != nil {
		return err
	}
	cfg.Posts, err = filepath.Abs(cfg.Posts)
	if err != nil {
		return err
	}
	cfg.Data, err = filepath.Abs(cfg.Data)
	if err != nil {
		return err
	}
	cfg.Includes, err = filepath.Abs(cfg.Includes)
	if err != nil {
		return err
	}
	cfg.Layouts, err = filepath.Abs(cfg.Layouts)
	if err != nil {
		return err
	}

	cfg.Source = filepath.ToSlash(cfg.Source)
	cfg.Destination = filepath.ToSlash(cfg.Destination)
	cfg.Posts = filepath.ToSlash(cfg.Posts)
	cfg.Data = filepath.ToSlash(cfg.Data)
	cfg.Includes = filepath.ToSlash(cfg.Includes)
	cfg.Layouts = filepath.ToSlash(cfg.Layouts)
	cfg.vars["site"] = pongo.Context{}
	return nil
}

func (cfg *config) toPageUrl(from string) string {
	return join(cfg.Baseurl, filepath.ToSlash(from[len(cfg.Source):]))
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

func (cfg *config) toPostUrl(from string, pageVars map[string]interface{}) string {
	ext := filepath.Ext(from)
	name := filepath.Base(from)
	name = name[0:len(name)-len(ext)] + ".html"
	if len(name) > 11 {
		date, err := time.Parse("2006-01-02-", name[:11])
		if err == nil {
			category := ""
			if v, ok := pageVars["category"]; ok {
				category, _ = v.(string)
			}
			title := name[11:]
			if v, ok := pageVars["title"]; ok {
				title, _ = v.(string)
			}
			postUrl := cfg.Permalink
			postUrl = strings.Replace(postUrl, ":categories", category, -1)
			postUrl = strings.Replace(postUrl, ":year", fmt.Sprintf("%d", date.Year()), -1)
			postUrl = strings.Replace(postUrl, ":month", fmt.Sprintf("%d", date.Month()), -1)
			postUrl = strings.Replace(postUrl, ":i_month", fmt.Sprintf("%02d", date.Month()), -1)
			postUrl = strings.Replace(postUrl, ":day", fmt.Sprintf("%d", date.Day()), -1)
			postUrl = strings.Replace(postUrl, ":i_day", fmt.Sprintf("%02d", date.Day()), -1)
			postUrl = strings.Replace(postUrl, ":title", title, -1)
			return join(cfg.Baseurl, postUrl)
		}
	}
	return join(cfg.Baseurl, name)
}

func (cfg *config) toPage(from string) string {
	if cfg.Permalink != "" {
	}
	return filepath.ToSlash(filepath.Join(cfg.Destination, from[len(cfg.Source):]))
}

func (cfg *config) toPost(from string, pageVars map[string]interface{}) string {
	ext := filepath.Ext(from)
	name := filepath.Base(from)
	name = name[0:len(name)-len(ext)] + ".html"
	if len(name) > 11 {
		date, err := time.Parse("2006-01-02-", name[:11])
		if err == nil {
			category := ""
			if v, ok := pageVars["category"]; ok {
				category, _ = v.(string)
			}
			title := name[11:]
			if v, ok := pageVars["title"]; ok {
				title, _ = v.(string)
			}
			postUrl := cfg.Permalink
			postUrl = strings.Replace(postUrl, ":categories", category, -1)
			postUrl = strings.Replace(postUrl, ":year", fmt.Sprintf("%d", date.Year()), -1)
			postUrl = strings.Replace(postUrl, ":month", fmt.Sprintf("%d", date.Month()), -1)
			postUrl = strings.Replace(postUrl, ":i_month", fmt.Sprintf("%02d", date.Month()), -1)
			postUrl = strings.Replace(postUrl, ":day", fmt.Sprintf("%d", date.Day()), -1)
			postUrl = strings.Replace(postUrl, ":i_day", fmt.Sprintf("%02d", date.Day()), -1)
			postUrl = strings.Replace(postUrl, ":title", title, -1)
			return filepath.ToSlash(filepath.Join(cfg.Destination, postUrl))
		}
	}
	return filepath.ToSlash(filepath.Join(cfg.Destination, name))
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
			date := cfg.toDate(src)
			pageUrl := cfg.toPostUrl(src, pageVars)
			title := str(vars["title"])
			vars["post"] = map[string]interface{}{
				"date":  date,
				"url":   pageUrl,
				"title": title,
			}
			vars["page"] = map[string]interface{}{
				"date":  date,
				"url":   pageUrl,
				"title": title,
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
			src = filepath.ToSlash(filepath.Join(cfg.Layouts, str(vars["layout"])+".html"))
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
	pages := []map[string]interface{}{}
	err = filepath.Walk(cfg.Source, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.Source {
			return err
		}

		from := filepath.ToSlash(name)
		dot := filepath.Base(name)[0]
		if info.IsDir() {
			if from == cfg.Destination || dot == '.' || dot == '_' {
				return filepath.SkipDir
			}
		} else {
			if dot != '.' && dot != '_' {
				vars := map[string]interface{}{}
				vars["path"] = from
				vars["url"] = cfg.toPageUrl(from)
				vars["date"] = info.ModTime()
				pages = append(pages, vars)
			}
		}
		return err
	})
	checkFatal(err)

	categories := pongo.Context{}
	posts := []pongo.Context{}
	err = filepath.Walk(cfg.Posts, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.Posts {
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
		vars["path"] = from
		vars["url"] = cfg.toPostUrl(from, vars)
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

	cfg.vars["site"].(pongo.Context)["baseurl"] = cfg.Baseurl
	cfg.vars["site"].(pongo.Context)["time"] = time.Now()
	cfg.vars["site"].(pongo.Context)["pages"] = pages
	cfg.vars["site"].(pongo.Context)["posts"] = posts
	cfg.vars["site"].(pongo.Context)["categories"] = categories
	cfg.vars["site"].(pongo.Context)["data"] = pongo.Context{}

	fis, err := ioutil.ReadDir(cfg.Data)
	if err == nil {
		for _, fi := range fis {
			ext := filepath.Ext(fi.Name())
			var data interface{}
			switch ext {
			case ".yaml", ".yml":
				b, err := ioutil.ReadFile(filepath.Join(cfg.Data, fi.Name()))
				if err != nil {
					return err
				}
				err = goyaml.Unmarshal(b, &data)
				if err == nil {
					name := fi.Name()
					name = name[0 : len(name)-len(ext)]
					cfg.vars["site"].(pongo.Context)["data"].(pongo.Context)[name] = data
				}
			}
		}
	}

	if _, err := os.Stat(cfg.Destination); err != nil {
		err = os.MkdirAll(cfg.Destination, 0755)
		checkFatal(err)
	}

	for _, page := range pages {
		from := page["path"].(string)
		to := cfg.toPage(from)
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		checkFatal(err)
	}

	for _, post := range posts {
		from := post["path"].(string)
		to := cfg.toPost(from, post)
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
	err = filepath.Walk(cfg.Source, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.Destination {
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
				if filepath.HasPrefix(from, cfg.Destination) {
					continue
				}
				to := ""

				vars := map[string]interface{}{}
				_, err = parseFile(from, vars)
				if err != nil {
					continue
				}
				if filepath.HasPrefix(from, cfg.Posts) {
					to = cfg.toPost(from, vars)
				} else if filepath.HasPrefix(from, cfg.Source) {
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
	return http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), http.FileServer(http.Dir(cfg.Destination)))
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
