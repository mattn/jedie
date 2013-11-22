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
	"os"
	"path/filepath"
	"strings"
)

var extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS

type config struct {
	baseUrl     string `yaml:"base-url"`
	source      string `yaml:"source"`
	destination string `yaml:"destination"`
	vars		map[string]interface{}
}

func str(s interface{}) string {
	if ss, ok := s.(string); ok {
		return ss
	}
	return ""
}

func (cfg *config) convertFile(src, dst string) error {
	var err error
	ext := filepath.Ext(src)
	switch ext {
	case ".yml", ".go", ".exe":
		return nil
	case ".html", ".md", ".mkd":
		vars := pongo.Context{"content": ""}
		for k, v := range cfg.vars {
			vars[k] = v
		}
		dst = dst[0:len(dst)-len(ext)] + ".html"

		for {
			b, err := ioutil.ReadFile(src)
			if err != nil {
				// TODO Really?
				break
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
			// FIXME Why pongo returns pointer of string?
			ps := new(string)
			*ps = content
			tpl, err := pongo.FromString(str(vars["layout"]), ps, nil)
			if err == nil {
				output, err := tpl.Execute(&vars)
				if err == nil && output != nil {
					content = *output
				}
			} else {
				return err
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
			src = filepath.Join(cfg.source, "_layouts", str(vars["layout"])+".html")
			ext = filepath.Ext(src)
			content = str(vars["content"])
			vars = pongo.Context{"content": content}
		}
		fmt.Println(dst)

		return ioutil.WriteFile(dst, []byte(str(vars["content"])), 0644)
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

	var globalVariables map[string]interface{}
	err = goyaml.Unmarshal(b, &globalVariables)
	if err != nil {
		log.Fatal(err)
	}

	var cfg config
	cfg.vars = globalVariables
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

	pongo.Filters["escape"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
        str, is_str := value.(string)
        if !is_str {
                return nil, errors.New(fmt.Sprintf("%v (%T) is not of type string", value, value))
        }
        return str, nil
	}

	err = filepath.Walk(cfg.source, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}

		from := filepath.ToSlash(path)
		to := filepath.Join(cfg.destination, from[len(cfg.source):])
		if info.IsDir() {
			dot := filepath.Base(path)[0]
			if from == cfg.destination || dot == '.' || dot == '_' {
				return filepath.SkipDir
			}
			err = os.MkdirAll(to, 0755)
		} else {
			err = cfg.convertFile(from, to)
		}
		fmt.Println(from, "=>", to)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
