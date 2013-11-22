package main

import (
	"flag"
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
	"os"
	"path/filepath"
)

type config struct {
	baseUrl     string `yaml:"base-url"`
	source      string `yaml:"source"`
	destination string `yaml:"destination"`
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
	var cfg config
	err = goyaml.Unmarshal(b, &cfg)
	if err != nil {
		log.Fatal(err)
	}
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
			_, err = copyFile(from, to)
		}
		log.Println(from, to)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
