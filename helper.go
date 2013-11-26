package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/flosch/pongo"
	"github.com/russross/blackfriday"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/url"
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

func str(s interface{}) string {
	if ss, ok := s.(string); ok {
		return ss
	}
	return ""
}

func include(cfg *config, vars pongo.Context) func(*string) (*string, error) {
	return func(loc *string) (*string, error) {
		inc := filepath.ToSlash(filepath.Join(cfg.includes, *loc))
		tpl, err := pongo.FromFile(inc, include(cfg, vars))
		if err != nil {
			return nil, err
		}
		return tpl.Execute(&vars)
	}
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
	pongo.Filters["xml_escape"] = func(value interface{}, args []interface{}, ctx *pongo.FilterChainContext) (interface{}, error) {
		str, is_str := value.(string)
		if !is_str {
			return nil, errors.New(fmt.Sprintf("%v (%T) is not of type string", value, value))
		}
		var b bytes.Buffer
		xml.Escape(&b, []byte(str))
		return b.String(), nil
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
			l := rv.Len()
			if l < limit {
				limit = l
			}
			return rv.Slice(0, l).Interface(), nil
		case reflect.String:
			return value, nil
		default:
			return nil, errors.New(fmt.Sprintf("Cannot join variable of type %T ('%v').", value, value))
		}
		panic("unreachable")
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
