package main

import (
	"flag"
	"fmt"
	"github.com/flosch/pongo"
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
	baseUrl     string `yaml:"baseUrl"`
	source      string `yaml:"source"`
	destination string `yaml:"destination"`
	posts       string `yaml:"posts"`
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
	cfg.baseUrl = str(globalVariables["baseUrl"])
	cfg.source = str(globalVariables["source"])
	cfg.destination = str(globalVariables["destination"])
	cfg.posts = str(globalVariables["posts"])
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
	cfg.includes = filepath.ToSlash(cfg.includes)
	cfg.layouts = filepath.ToSlash(cfg.layouts)
	cfg.vars["site"] = pongo.Context{}
	return nil
}

func (cfg *config) toPageUrl(from string) string {
	return cfg.baseUrl + filepath.ToSlash(from[len(cfg.source):])
}

func (cfg *config) toPostUrl(from string) string {
	ext := filepath.Ext(from)
	name := filepath.Base(from)
	name = name[0:len(name)-len(ext)] + ".html"
	if len(name) > 11 {
		date, err := time.Parse("2006-01-02-", name[:11])
		if err == nil {
			return cfg.baseUrl + date.Format("/2006/01/02/") + name[11:]
		}
	}
	return cfg.baseUrl + "/" + name
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
			return filepath.ToSlash(filepath.Join(cfg.destination, date.Format("/2006/01/02/") + name[11:]))
		}
	}
	return filepath.ToSlash(filepath.Join(cfg.destination, name))
}

func (cfg *config) convertFile(src, dst string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}
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
				"url": cfg.toPostUrl(src),
				"title": str(vars["title"]),
			}
			vars["page"] = pongo.Context{
				"date": fi.ModTime(),
				"url": cfg.toPostUrl(src),
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
			src = filepath.ToSlash(filepath.Join(cfg.layouts, str(vars["layout"])+".html"))
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
	err = ioutil.WriteFile(filepath.Join(p, "_posts", time.Now().Format("2006-01-02-welcome-to-jedie.md")), []byte(postsBlog), 0644)
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
		} else {
			if dot != '.' && dot != '_' {
				pageFiles = append(pageFiles, from)
				vars := pongo.Context{}
				vars["url"] = cfg.toPageUrl(from)
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
	err = filepath.Walk(cfg.posts, func(name string, info os.FileInfo, err error) error {
		if info == nil || name == cfg.posts {
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
		to := cfg.toPost(from)
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
