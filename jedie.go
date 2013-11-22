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
	"os"
	"path"
	"path/filepath"
	"strings"
	//"time"
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

/*
type post struct {
	title string
	file string
	vars        pongo.Context
}
*/

/* TODO
type site struct {
	time time.Time
	pages []string
	posts []post
}
*/

func str(s interface{}) string {
	if ss, ok := s.(string); ok {
		return ss
	}
	return ""
}

func (cfg *config) toUrl(from string) string {
	return cfg.baseUrl + path.Clean(from[len(cfg.source):])
}

func (cfg *config) toPage(from string) string {
	return filepath.ToSlash(filepath.Join(cfg.destination, from[len(cfg.source):]))
}

func (cfg *config) toPost(from string) string {
	base := filepath.ToSlash(filepath.Join(cfg.source, "_posts"))
	return filepath.ToSlash(filepath.Join(cfg.destination, from[len(base):]))
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
		for n, line = range lines[1:] {
			if line == "---" {
				break
			}
			token := strings.SplitN(line, ":", 2)
			if len(token) == 2 && token[0] != "" {
				vars[strings.TrimSpace(token[0])] = strings.TrimSpace(token[1])
			}
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

func (cfg *config) convertFile(src, dst string) error {
	var err error
	ext := filepath.Ext(src)
	switch ext {
	case ".yml", ".go", ".exe":
		return nil
	case ".html", ".md", ".mkd":
		dst = dst[0:len(dst)-len(ext)] + ".html"
		fi, err := os.Stat(src)
		if err != nil {
			return err
		}

		vars := pongo.Context{"content": ""}
		for {
			for k, v := range cfg.vars {
				vars[k] = v
			}
			content, err := parseFile(src, vars)
			if err != nil {
				// TODO Really?
				break
			}
			vars["post"] = pongo.Context{
				"date": fi.ModTime(),
				"url": cfg.toUrl(src),
				"title": vars["title"],
			}
			// FIXME Why pongo returns pointer of string?
			if content != "" {
				ps := new(string)
				*ps = content
				//old := cfg.vars
				//cfg.vars = vars
				tpl, err := pongo.FromString(str(vars["layout"]), ps, include(cfg, vars))
				//cfg.vars = old
				if err == nil {
					output, err := tpl.Execute(&vars)
					if err == nil && output != nil {
						content = *output
					} else {
					println(err.Error())
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
			if vars["layout"] == "" {
				break
			}
			src = filepath.ToSlash(filepath.Join(cfg.source, "_layouts", str(vars["layout"])+".html"))
			ext = filepath.Ext(src)
			content = str(vars["content"])
			vars = pongo.Context{"content": content}
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

func main() {
	flag.Parse()

	b, err := ioutil.ReadFile("_config.yml")
	if err != nil {
		log.Fatal(err)
	}

	var globalVariables pongo.Context
	err = goyaml.Unmarshal(b, &globalVariables)
	if err != nil {
		log.Fatal(err)
	}

	var cfg config
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
		log.Fatal(err)
	}
	cfg.destination, err = filepath.Abs(cfg.destination)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(cfg.destination); err != nil {
		err = os.MkdirAll(cfg.destination, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	cfg.source = filepath.ToSlash(cfg.source)
	cfg.destination = filepath.ToSlash(cfg.destination)
	cfg.vars["site"] = pongo.Context{}

	pongo.Filters["escape"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		str, is_str := value.(string)
		if !is_str {
			return nil, errors.New(fmt.Sprintf("%v (%T) is not of type string", value, value))
		}
		return url.QueryEscape(str), nil
	}

	var pages []string
	err = filepath.Walk(cfg.source, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}

		from := filepath.ToSlash(path)
		if info.IsDir() {
			dot := filepath.Base(path)[0]
			if from == cfg.destination || dot == '.' || dot == '_' {
				return filepath.SkipDir
			}
			err = os.MkdirAll(cfg.toPage(from), 0755)
		} else {
			pages = append(pages, from)
		}
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	categories := pongo.Context{}
	var postFiles []string
	var posts []pongo.Context
	base := filepath.ToSlash(filepath.Join(cfg.source, "_posts"))
	err = filepath.Walk(base, func(name string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}

		if !info.IsDir() {
			from := filepath.ToSlash(name)
			vars := pongo.Context{}
			_, err = parseFile(from, vars)
			if err != nil {
				return err
			}
			vars["url"] = cfg.toUrl(from)
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
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	cfg.vars["site"].(pongo.Context)["pages"] = pages
	cfg.vars["site"].(pongo.Context)["posts"] = posts
	cfg.vars["site"].(pongo.Context)["categories"] = categories

	for _, from := range pages {
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
}
