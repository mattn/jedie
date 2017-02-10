package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/flosch/pongo2"
	"github.com/russross/blackfriday"
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

func pongoSetup() {
	pongo2.ReplaceFilter("safe", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		output := strings.Replace(in.String(), "&", "&amp;", -1)
		output = strings.Replace(output, ">", "&gt;", -1)
		output = strings.Replace(output, "<", "&lt;", -1)
		output = strings.Replace(output, "\"", "&quot;", -1)
		output = strings.Replace(output, "'", "&#39;", -1)
		return pongo2.AsValue(output), nil
	})
	pongo2.ReplaceFilter("escape", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return in, nil
	})
	pongo2.RegisterFilter("xml_escape", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		var b bytes.Buffer
		xml.Escape(&b, []byte(in.String()))
		return pongo2.AsValue(b.String()), nil
	})
	pongo2.RegisterFilter("truncate", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		rs := []rune(in.String())
		letters := param.Integer()
		if letters > len(rs) {
			letters = len(rs)
		}
		return pongo2.AsValue(string(rs[:letters])), nil
	})
	pongo2.RegisterFilter("strip_html", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return pongo2.AsValue(regexp.MustCompile("<[^>]+>").ReplaceAllString(in.String(), "")), nil
	})
	pongo2.RegisterFilter("strip_newlines", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return pongo2.AsValue(strings.Replace(in.String(), "\n", "", -1)), nil
	})
	pongo2.RegisterFilter("date_to_string", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		date, ok := in.Interface().(time.Time)
		if !ok {
			return nil, &pongo2.Error{
				Sender:   "date_to_string",
				ErrorMsg: fmt.Sprintf("Date must be of type time.Time not %T ('%v')", in, in),
			}
		}
		return pongo2.AsValue(date.Format("2006/01/02 15:04:05")), nil
	})
	pongo2.RegisterFilter("date_to_rfc822", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		date, ok := in.Interface().(time.Time)
		if !ok {
			return nil, &pongo2.Error{
				Sender:   "date_to_rfc822",
				ErrorMsg: fmt.Sprintf("Date must be of type time.Time not %T ('%v')", in, in),
			}
		}
		return pongo2.AsValue(date.Format(time.RFC822)), nil
	})
	pongo2.ReplaceFilter("date", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		date, ok := in.Interface().(time.Time)
		if !ok {
			return nil, &pongo2.Error{
				Sender:   "date",
				ErrorMsg: fmt.Sprintf("Date must be of type time.Time not %T ('%v')", in, in),
			}
		}
		format := param.String()

		replacements := []struct {
			from string
			to   string
		}{
			{"%a", "Mon"},
			{"%A", "Monday"},
			{"%b", "Jan"},
			{"%B", "January"},
			{"%c", time.RFC3339},
			{"%C", "06"},
			{"%d", "02"},
			{"%C", "01/02/06"},
			{"%e", "_1/_2/_6"},
			// {"%E", ""},
			{"%F", "06-01-02"},
			// {"%G", ""},
			// {"%g", ""},
			{"%h", "Jan"},
			{"%H", "15"},
			{"%I", "03"},
			// {"%j", ""},
			{"%k", "3"},
			{"%l", "_3"},
			{"%m", "01"},
			{"%M", "04"},
			{"%n", "\n"},
			// {"%O", ""},
			{"%p", "PM"},
			{"%P", "pm"},
			{"%r", "03:04:05 PM"},
			{"%R", "03:04"},
			// {"%s", ""},
			{"%S", "05"},
			{"%t", "\t"},
			{"%T", "15:04:05"},
			// {"%u", ""},
			// {"%U", ""},
			// {"%V", ""},
			// {"%W", ""},
			// {"%x", ""},
			// {"%X", ""},
			{"%y", "06"},
			{"%Y", "2006"},
			{"%z", "-0700"},
			{"%Z", "MST"},
			// {"%+", ""},
			{"%%", "%"},
		}

		for _, replacement := range replacements {
			format = strings.Replace(format, replacement.from, replacement.to, -1)
		}

		return pongo2.AsValue(date.Format(format)), nil
	})
	pongo2.RegisterFilter("limit", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		limit := param.Integer()
		switch {
		case in.CanSlice():
			l := in.Len()
			if l < limit {
				limit = l
			}
			return in.Slice(0, limit), nil
		case in.IsString():
			l := in.Len()
			if l < limit {
				limit = l
			}
			return pongo2.AsValue(in.String()[:limit]), nil
		default:
			return nil, &pongo2.Error{
				Sender:   "limit",
				ErrorMsg: fmt.Sprintf("Cannot join variable of type %T ('%v').", in, in),
			}
		}
	})
	pongo2.RegisterFilter("prepend", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		input := strings.Replace(in.String(), "\n", "", -1)
		if input == "" {
			return pongo2.AsValue(""), nil
		}
		u, e := url.Parse(input)
		if e == nil && u.Host != "" {
			return pongo2.AsValue(input), nil
		}
		base := param.String()
		b, e := url.Parse(base)
		if e != nil {
			return nil, &pongo2.Error{
				Sender:   "prepend",
				ErrorMsg: fmt.Sprintf("Cannot prepend string ('%v').", param),
			}
		}
		b.Path = input
		return pongo2.AsValue(b.String()), nil
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

func urlJoin(l, r string) string {
	r = path.Clean(r)
	ls := strings.HasSuffix(l, "/")
	rp := strings.HasPrefix(r, "/")

	if ls && rp {
		return l + r[1:]
	}
	if !ls && !rp {
		return l + "/" + r
	}
	return l + r
}
