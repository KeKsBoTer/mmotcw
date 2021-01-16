package main

import (
	"flag"
	"fmt"
	"html/template"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	logger "github.com/withmandala/go-log"
)

var log *logger.Logger

func init() {
	log = logger.New(os.Stdout).WithColor()
}

// GetMaimais returns all maiamis for a given year structured in weeks
func GetMaimais(source MaimaiSource, year int) ([]Week, error) {
	weekFolders, err := filepath.Glob(filepath.Join(string(source), strconv.Itoa(year), "CW_*"))
	if err != nil {
		return nil, err
	}
	weeks := make([]Week, len(weekFolders))
	for i, w := range weekFolders {
		week, err := ReadWeek(w)
		if err != nil {
			return nil, err
		}
		weeks[i] = *week
	}
	// sort weeks
	sort.Slice(weeks[:], func(i, j int) bool {
		return weeks[j].CW.Before(weeks[i].CW)
	})
	return weeks, nil
}

func index(template template.Template, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _, _ := r.BasicAuth()
		year := getYear(r)
		maimais, err := GetMaimais(source, year)
		if err != nil {
			log.Error(err)
			return
		}
		err = template.Execute(w, struct {
			Weeks []Week
			User  string
		}{
			Weeks: maimais,
			User:  user,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func userContent(template template.Template, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := mux.Vars(r)["user"]
		year := getYear(r)
		weeks, err := GetMaimais(source, year)
		if err != nil {
			httpError(w, http.StatusInternalServerError)
			log.Error(err)
			return
		}
		empty := true
		for w := range weeks {
			filtered := []Maimai{}
			for _, m := range weeks[w].Maimais {
				creator := strings.ToLower(strings.Trim(strings.Split(m.File, ".")[0], "_0123456789"))
				if creator == user {
					filtered = append(filtered, m)
					empty = false
				}
			}
			weeks[w].Maimais = filtered
		}
		if empty {
			httpError(w, http.StatusNotFound)
			return
		}

		err = template.Execute(w, struct {
			Weeks []Week
			User  string
		}{
			Weeks: weeks,
			User:  user,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func week(template template.Template, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		week, _ := strconv.Atoi(mux.Vars(r)["week"])

		year := getYear(r)

		maimais, err := source.GetMaimaisForCW(CW{Year: year, Week: week})
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		err = template.Execute(w, struct {
			Maimais Week
			Week    int
		}{
			Maimais: *maimais,
			Week:    week,
		})
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

func createRouter(templates *template.Template, source MaimaiSource) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", faviconHandler)

	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// file server for maimais
	fsMaimais := http.FileServer(http.Dir(string(source)))
	r.PathPrefix("/mm/").Handler(http.StripPrefix("/mm/", fsMaimais))

	r.HandleFunc("/", index(*templates.Lookup("index.html"), source))

	r.HandleFunc("/{user:[a-z]+}", userContent(*templates.Lookup("user.html"), source))

	r.HandleFunc("/CW_{week:[0-9]+}", week(*templates.Lookup("week.html"), source))

	return r
}

func readFlags() (string, int) {
	var directory = flag.String("dir", ".", "the maimai directory")
	var port = flag.Int("port", 8080, "port to run on")
	flag.Parse()
	return *directory, *port
}

// loadTemplates reads all .html files as tempaltes from given directory
func loadTemplates(dir string) *template.Template {

	funcMap := template.FuncMap{
		"numvotes": func(maimais []Maimai) []int {
			v := voteCount(len(maimais))
			votes := make([]int, v)
			for i := range votes {
				votes[i] = i
			}
			return votes
		},
		"add": func(a, b int) int {
			return a + b
		},
		"formatCW": func(cw int) string {
			return fmt.Sprintf("CW_%02d", cw)
		},
		"pathPrefix": func(s string) string {
			return filepath.Join("mm", s)
		},
	}

	return template.Must(template.New("templates").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html")))

}

func main() {
	miamaiDir, port := readFlags()

	ImgCache.dir = miamaiDir

	templates := loadTemplates("./templates")

	source := MaimaiSource(miamaiDir)
	router := createRouter(templates, source)
	http.Handle("/", router)

	log.Info("starting webserver on port", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}
