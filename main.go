package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"text/template"
	"time"

	"regexp"
)

type Maimai struct {
	User string
	Href string
	Time time.Time
}

type Week struct {
	Maimais []Maimai
	KW      int
	Result  string
}

var fre = regexp.MustCompile(`CW_\d{2}`)

func getMaimais(baseDir string, pathPrefix string) ([]Week, error) {
	weekFolders, err := filepath.Glob(filepath.Join(baseDir, "CW_*"))
	if err != nil {
		return nil, err
	}
	weeks := make([]Week, len(weekFolders))
	for i, w := range weekFolders {

		cw, err := strconv.Atoi(filepath.Base(w)[3:])
		if err != nil {
			return nil, err
		}
		imgFiles, err := ioutil.ReadDir(w)
		if err != nil {
			return nil, err
		}
		week := Week{
			Maimais: []Maimai{},
			KW:      cw,
		}
		for _, img := range imgFiles {
			if !img.IsDir() {
				switch filepath.Ext(img.Name())[1:] {
				case
					"jpg",
					"jpeg",
					"gif",
					"png":
					week.Maimais = append(week.Maimais, Maimai{
						User: img.Name(),
						Href: filepath.Join(pathPrefix, filepath.Base(w), img.Name()),
						Time: img.ModTime(),
					})
					break
				case "html":
					abs := filepath.Join(w, img.Name())
					content, err := ioutil.ReadFile(abs)
					if err != nil {
						return nil, err
					}
					week.Result = string(content)
				}
			}
		}
		sort.Slice(week.Maimais[:], func(i, j int) bool {
			return week.Maimais[i].Time.After(week.Maimais[j].Time)
		})
		weeks[i] = week
	}
	sort.Slice(weeks[:], func(i, j int) bool {
		return weeks[i].KW > weeks[j].KW
	})
	return weeks, nil
}

var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	var directory = flag.String("dir", ".", "the maimai directory")
	var port = flag.Int("port", 8080, "port to run on")
	flag.Parse()

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalln(err)
	}

	index := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - not found"))
			return
		}
		tmpl, err = template.ParseFiles("templates/index.html")
		maimais, err := getMaimais(*directory, "mm")
		if err != nil {
			log.Fatalln(err)
		}
		err = tmpl.Execute(w, maimais)
		if err != nil {
			fmt.Println(err)
		}
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fs2 := http.FileServer(http.Dir(*directory))
	http.Handle("/mm/", http.StripPrefix("/mm/", fs2))

	http.HandleFunc("/", index)

	if err := http.ListenAndServe(":"+strconv.Itoa(*port), nil); err != nil {
		log.Fatalln(err)
	}
}
