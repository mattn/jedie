package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS

func str(s interface{}) string {
	if ss, ok := s.(string); ok {
		return ss
	}
	return ""
}

func join(l, r string) string {
	if strings.HasSuffix(l, "/") && strings.HasPrefix(r, "/") {
		return l + r[1:]
	}
	if !strings.HasSuffix(l, "/") && !strings.HasPrefix(r, "/") {
		return l + "/" + r
	}
	return strings.Replace(l+r, "//", "/", -1)
}

func include(cfg *config, vars pongo2.Context) func(*string) (string, error) {
	return func(loc *string) (string, error) {
		inc := filepath.ToSlash(filepath.Join(cfg.Includes, *loc))
		tpl, err := pongo2.FromFile(inc)
		if err != nil {
			return "", err
		}
		newvars := pongo2.Context{}
		newvars.Update(cfg.vars)
		newvars.Update(vars)
		return tpl.Execute(newvars)
	}
}

func parseFile(file string, vars pongo2.Context) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	content := string(b)
	lines := strings.Split(content, "\n")
	if len(lines) > 2 && lines[0] == "---" {
		var n int
		var line string
		for n, line = range lines[1:] {
			if line == "---" {
				break
			}
		}
		err = yaml.Unmarshal(b, &vars)
		if err != nil {
			return "", err
		}
		content = strings.Join(lines[n+2:], "\n")
	} else if isMarkdown(file) {
		vars["title"] = ""
		vars["layout"] = "plain"
		vars["date"] = ""
	}
	return content, nil
}

func pongoSetup() {
	pongo2.ReplaceFilter("safe", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		return in, nil
	})
	pongo2.RegisterFilter("xml_escape", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		var b bytes.Buffer
		xml.Escape(&b, []byte(in.String()))
		return pongo2.AsValue(b.String()), nil
	})
	pongo2.RegisterFilter("truncate", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		rs := []rune(in.String())
		letters := param.Integer()
		if letters > len(rs) {
			letters = len(rs)
		}
		return pongo2.AsValue(string(rs[:letters])), nil
	})
	pongo2.RegisterFilter("strip_html", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		return pongo2.AsValue(regexp.MustCompile("<[^>]+>").ReplaceAllString(in.String(), "")), nil
	})
	pongo2.RegisterFilter("date_to_string", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		date, ok := in.Interface().(time.Time)
		if !ok {
			return nil, errors.New(fmt.Sprintf("Date must be of type time.Time not %T ('%v')", in, in))
		}
		return pongo2.AsValue(date.Format("2006/01/02 03:04:05")), nil
	})
	pongo2.RegisterFilter("limit", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err error) {
		limit := param.Integer()
		switch {
		case in.CanSlice():
			l := in.Len()
			if l < limit {
				limit = l
			}
			return in.Slice(0, l), nil
		case in.IsString():
			l := in.Len()
			if l < limit {
				limit = l
			}
			return pongo2.AsValue(in.String()[:l]), nil
		default:
			return nil, errors.New(fmt.Sprintf("Cannot join variable of type %T ('%v').", in, in))
		}
		panic("unreachable")
	})
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

func isMarkdown(src string) bool {
	ext := filepath.Ext(src)
	switch ext {
	case ".md", ".mkd", ".markdown":
		return true
	}
	return false
}

func isConvertable(src string) bool {
	ext := filepath.Ext(src)
	switch ext {
	case ".html", ".xml", ".md", ".mkd", ".markdown":
		return true
	}
	return false
}
