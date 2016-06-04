# Jedie

[![GoDoc](https://godoc.org/github.com/mattn/jedie?status.svg)](https://godoc.org/github.com/mattn/jedie)
[![Go Report Card](https://goreportcard.com/badge/github.com/mattn/jedie)](https://goreportcard.com/report/github.com/mattn/jedie)

         __       ___    
     __ / /__ ___/ (_)__
    / // / -_) _  / / -_)
    L___/`__/`_,_/_/`__/

jedie - static site generator, jekyll replacement, in golang.

## Install

### Requirements

*   golang (of course!)
*   git

### Install with `go get`

```
$ go get github.com/mattn/jedie
```

### Or, Build after git clone

Get dependencies at first.

```
git clone https://github.com/mattn/jedie
cd jedie
go get github.com/flosch/pongo2
go get github.com/howeyc/fsnotify
go get github.com/russross/blackfriday
go get gopkg.in/yaml.v1
go build
```

## Usage

At the first, create scaffold

```
$ mkdir /path/to/blog
$ jedie new /path/to/blog
$ cd /path/to/blog
$ vim _posts/2013-11-23-welcome-to-jedie.md
$ jedie build
```

Then, you can see your site is built in `_site` directory.
If you want to serve your site with http server:

```
$ jedie serve
```

## Configuration

For example, you can do your specified conversion like below.

```yaml
conversion:
  js:
    ext: js
    command: minifyjs -m -i {{from}} -o {{to}}
```

## Author

Yasuhiro Matsumoto (a.k.a mattn)

## License

MIT
