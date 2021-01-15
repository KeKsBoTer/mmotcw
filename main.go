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
	"time"

	"github.com/gorilla/mux"
	logger "github.com/withmandala/go-log"
)

var log *logger.Logger

func init() {
	log = logger.New(os.Stdout).WithColor()
}

func getMaimais(baseDir string) ([]Week, error) {
	weekFolders, err := filepath.Glob(filepath.Join(baseDir, "CW_*"))
	if err != nil {
		return nil, err
	}
	weeks := make([]Week, len(weekFolders))
	for i, w := range weekFolders {

		week, err := getMaiMaiPerCW("mm", w)

		if err != nil {
			return nil, err
		}
		uploadLock := CheckLock("upload", w)
		voteLock := CheckLock("vote", w)

		templateFiles, err := filepath.Glob(filepath.Join(w, "template.*"))

		if err == nil && len(templateFiles) > 0 {
			week.Template = filepath.Join("mm", filepath.Base(w), filepath.Base(templateFiles[0]))
		}

		cw, _ := strconv.Atoi(filepath.Base(w)[3:])
		if voteLock && uploadLock {
			votes, err := getVoteResults(w, cw)
			if err == nil {
				week.Votes = votes
			} else {
				log.Error(err)
			}
		}
		week.CanVote = uploadLock && !voteLock

		weeks[i] = *week
	}
	sort.Slice(weeks[:], func(i, j int) bool {
		return weeks[j].CW.Before(weeks[i].CW)
	})
	return weeks, nil
}

func index(template template.Template, directory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _, _ := r.BasicAuth()
		var year int
		// change year if year is given
		if y, ok := mux.Vars(r)["year"]; ok {
			yy, _ := strconv.Atoi(y)
			year = yy
		} else {
			year = time.Now().Year()
		}
		maimais, err := getMaimais(filepath.Join(directory, strconv.Itoa(year)))
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

func userContent(template template.Template, directory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := mux.Vars(r)["user"]
		weeks, err := getMaimais(directory)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - server ist kaputt"))
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
			weeks[w].CanVote = false
			weeks[w].Votes = nil
		}
		if empty {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 page not found"))
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

func week(template template.Template, directory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		week, _ := strconv.Atoi(mux.Vars(r)["week"])

		// get year or use default (current year)
		var year int
		y, ok := r.URL.Query()["year"]
		if !ok || len(y[0]) < 1 {
			year = time.Now().Year()
		} else if yy, err := strconv.Atoi(y[0]); err == nil {
			year = yy
		} else {
			year = time.Now().Year()
		}

		cwPath := filepath.Join(directory, strconv.Itoa(year), fmt.Sprintf("CW_%02d", week))
		if _, err := os.Stat(cwPath); err != nil {
			httpError(w, http.StatusNotFound)
			return
		}
		maimais, err := getMaiMaiPerCW("mm", cwPath)
		if err != nil {
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

func createRouter(templates *template.Template, maimaiDir string) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", faviconHandler)

	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// file server for maimais
	fsMaimais := http.FileServer(http.Dir(maimaiDir))
	r.PathPrefix("/mm/").Handler(http.StripPrefix("/mm/", fsMaimais))

	r.HandleFunc("/", index(*templates.Lookup("index.html"), maimaiDir))

	r.HandleFunc("/{user:[a-z]+}", userContent(*templates.Lookup("user.html"), maimaiDir))

	r.HandleFunc("/CW_{week:[0-9]+}", week(*templates.Lookup("week.html"), maimaiDir))

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
	}

	return template.Must(template.New("templates").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html")))

}

func main() {
	miamaiDir, port := readFlags()

	templates := loadTemplates("./templates")

	router := createRouter(templates, miamaiDir)
	http.Handle("/", router)

	log.Info("starting webserver on port", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}
