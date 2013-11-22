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
			content, err := parseFile(src, vars)
			if err != nil {
				return err
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
			if str(vars["layout"]) == "" {
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

func pongoSetup() {
	pongo.Filters["escape"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		str, is_str := value.(string)
		if !is_str {
			return nil, errors.New(fmt.Sprintf("%v (%T) is not of type string", value, value))
		}
		return url.QueryEscape(str), nil
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

	pongoSetup()

	var pageFiles []string
	pages := []pongo.Context{}
	err = filepath.Walk(cfg.source, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}

		from := filepath.ToSlash(path)
		dot := filepath.Base(path)[0]
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
		if info == nil {
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
		to := filepath.Join(cfg.destination, from[len(base):])
		to = to[0:len(to)-len(ext)] + ".html"
		fmt.Println(from, "=>", to)
		err = cfg.convertFile(from, to)
		if err != nil {
			log.Fatal(err)
		}
	}

	if flag.NArg() != 1 && flag.Arg(0) == "server" {
		http.ListenAndServe(":4000", http.FileServer(http.Dir("_site")))
	}
}
