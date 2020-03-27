package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"regexp"
	"time"

	"strconv"

	"golang.org/x/net/html"
)

// File is a file :D
type File struct {
	name string
	dir  bool
	time time.Time
}

var dateRe = regexp.MustCompile(`\s*\d{2}-\w{3}-\d{4} \d{2}:\d{2}\s+\d+`)

func getFiles(url string) ([]File, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	z := html.NewTokenizer(res.Body)

	files := []File{}
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			sort.Slice(files[:], func(i, j int) bool {
				return files[i].time.After(files[j].time)
			})
			return files, nil
		case tt == html.TextToken:
			t := z.Token()
			if dateRe.Match([]byte(t.Data)) {
				date := strings.Split(strings.Trim(t.Data, " "), "  ")[0]
				files[len(files)-1].time, err = time.Parse("02-Jan-2006 15:04", date)
				if err != nil {
					fmt.Println(err)
				}
			}

		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a"
			if isAnchor {
				for _, a := range t.Attr {
					if a.Key == "href" && !strings.HasPrefix(a.Val, "..") {
						files = append(files, File{
							name: a.Val,
							dir:  strings.HasSuffix(a.Val, "/"),
						})
						break
					}
				}
			}
		}
	}
}

type Maimai struct {
	User string
	Href string
}

type Week struct {
	Maimais []Maimai
	KW      int
}

var fre = regexp.MustCompile(`CW_\d{2}`)

func getMaimais(baseUrl string) ([]Week, error) {
	weeks := []Week{}
	baseFiles, err := getFiles(baseUrl)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	for _, w := range baseFiles {
		if w.dir && fre.Match([]byte(w.name[:5])) {
			kw, _ := strconv.Atoi(w.name[3:5])
			week := Week{Maimais: []Maimai{}, KW: kw}
			maimais, err := getFiles(baseUrl + w.name)
			if err != nil {
				return nil, err
			}
			for _, m := range maimais {
				if !m.dir && (strings.HasSuffix(m.name, ".png") || strings.HasSuffix(m.name, ".gif") || strings.HasSuffix(m.name, ".jpg") || strings.HasSuffix(m.name, ".jpeg")) {
					week.Maimais = append(week.Maimais, Maimai{
						User: m.name,
						Href: baseUrl + w.name + m.name,
					})
				}
			}
			weeks = append(weeks, week)
		}
	}
	return weeks, nil
}

func main() {

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatalln(err)
	}

	index := func(w http.ResponseWriter, r *http.Request) {
		maimais, err := getMaimais("https://marg.selfhost.co/mmotcw/")
		if err != nil {
			log.Fatalln(err)
		}
		err = tmpl.Execute(w, maimais)
		if err != nil {
			fmt.Println(err)
		}
	}

	http.HandleFunc("/", index)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalln(err)
	}
}
