package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/rand"
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

func index(template template.Template, source MaimaiSource, s *Subscriptions) http.HandlerFunc {
	fq, _ := os.Open("templates/quotes.txt")
	defer fq.Close()
	q, _ := io.ReadAll(fq)
	quotes := strings.Split(string(q), "\n")

	users := []string{"Simon", "Matthis", "Fabio", "Jannis", "Lena", "Christian", "Daniel", "Lorenz"}

	return func(w http.ResponseWriter, r *http.Request) {

		user, _, _ := r.BasicAuth()
		year := getYear(r)
		maimais, err := GetMaimais(source, year)
		if err != nil {
			log.Error(err)
			return
		}

		if log.IsDebug() {
			template = *loadTemplates("templates").Lookup("index.html")
		}

		otherYear := 2020
		if year == 2020 {
			otherYear = 2021
		}

		rand.Seed(int64(time.Now().Day()))

		err = template.Execute(w, struct {
			Weeks         []Week
			User          string
			PushPublicKey string
			Year          int
			OtherYear     int
			Quote         string
			QuoteAuthor   string
		}{
			Weeks:         maimais,
			User:          user,
			PushPublicKey: s.publicKey,
			Year:          year,
			OtherYear:     otherYear,
			Quote:         quotes[rand.Intn(len(quotes))],
			QuoteAuthor:   users[rand.Intn(len(users))],
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
			filtered := []UserMaimai{}
			for _, m := range weeks[w].Maimais {
				if strings.EqualFold(string(m.User), user) {
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
		year, _ := strconv.Atoi(mux.Vars(r)["year"])

		maimais, err := source.GetMaimaisForCW(CW{Year: year, Week: week})
		if err != nil {
			switch err.(type) {
			case *os.PathError:
				httpError(w, http.StatusNotFound)
			default:
				log.Error(err)
				httpError(w, http.StatusInternalServerError)
			}
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

func year(template template.Template, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year, _ := strconv.Atoi(mux.Vars(r)["year"])
		CWs, err := source.GetCWsOfYear(year)
		if err != nil {
			switch err.(type) {
			case *os.PathError:
				httpError(w, http.StatusNotFound)
			default:
				log.Error(err)
				httpError(w, http.StatusInternalServerError)
			}
			return
		}

		firstCW := CWs[0]
		cwFiller := make([]CW, firstCW.Week%10)
		cwFiller = append(cwFiller, CWs...)

		const columns = 10
		l := len(cwFiller) / columns
		if len(cwFiller)%columns != 0 {
			l++
		}
		splitCw := make([][]CW, l)
		for i := range splitCw {
			splitCw[i] = cwFiller[i*columns : min((i+1)*columns, len(cwFiller)-1)]
		}

		err = template.Execute(w, struct {
			Year  int
			Weeks [][]CW
		}{
			Year:  year,
			Weeks: splitCw,
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

func createRouter(templates *template.Template, source MaimaiSource, sub *Subscriptions) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", faviconHandler)

	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// file server for maimais
	fsMaimais := http.FileServer(http.Dir(string(source)))
	r.PathPrefix("/mm/").Handler(http.StripPrefix("/mm/", fsMaimais))

	r.HandleFunc("/", index(*templates.Lookup("index.html"), source, sub))

	r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/js/sw.js")
	})

	r.HandleFunc("/vote", vote(source))

	r.HandleFunc("/upload", uploadHandler(source, sub))

	r.PathPrefix("/admin").Handler(adminRouter(*templates.Lookup("admin.html"), source))

	r.HandleFunc("/subscribe", subscribe(sub))

	r.HandleFunc("/{user:[a-z]+}", userContent(*templates.Lookup("user.html"), source))

	r.HandleFunc("/{year:202[0-9]}/", year(*templates.Lookup("year.html"), source))

	r.HandleFunc("/{year:202[0-9]}/CW_{week:[0-9]+}/", week(*templates.Lookup("week.html"), source))

	return r
}

func readFlags() (string, int, string, bool) {
	var directory = flag.String("dir", ".", "the maimai directory")
	var port = flag.Int("port", 8080, "port to run on")
	var subsDir = flag.String("subsdir", "/var/lib/mmotcw", "directory containing subscriptions, pub and priv-key")
	var noCacheInit = flag.Bool("no-cache-init", false, "Don't initialize image cache")
	flag.Parse()
	return *directory, *port, *subsDir, *noCacheInit
}

// loadTemplates reads all .html files as templates from given directory
func loadTemplates(dir string) *template.Template {

	funcMap := template.FuncMap{
		"numVotes": func(maimais []UserMaimai) []int {
			v := voteCount(len(maimais))
			votes := make([]int, v)
			for i := range votes {
				votes[i] = i
			}
			return votes
		},
		"add": func(a, b int) string {
			return fmt.Sprintf("%02d", a+b)
		},
		"formatCW": func(cw int) string {
			return fmt.Sprintf("CW_%02d", cw)
		},
		"pathPrefix": func(s string) string {
			return filepath.Join("mm", s)
		},
		"toJson": func(i interface{}) string {
			b, err := json.Marshal(i)
			if err != nil {
				log.Error(err)
				return ""
			}
			return base64.RawStdEncoding.EncodeToString(b)
		},
		"formatTime": func(t time.Time) string {
			w := t.Weekday()

			weekdays := []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
			return fmt.Sprintf("%s %s", weekdays[w], t.Format("15:03"))
		},
	}

	return template.Must(template.New("templates").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html")))

}

func main() {
	// enable debug
	if os.Getenv("DEBUG") == "true" {
		log = log.WithDebug()
	}
	miamaiDir, port, subsDir, skipCacheInit := readFlags()

	ImgCache.dir = miamaiDir

	sub, err := ReadSubscriptions(
		subsDir+"/sub_key",
		subsDir+"/sub_key.pub",
		subsDir+"/subscriptions",
	)
	if err != nil {
		log.Fatal(err)
	}

	templates := loadTemplates("./templates")

	source := MaimaiSource(miamaiDir)
	router := createRouter(templates, source, sub)
	http.Handle("/", router)

	if !skipCacheInit {
		go func() {
			err = InitCache(source)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	serveOn := fmt.Sprintf("localhost:%d", port)
	log.Infof("starting webserver on http://%s", serveOn)
	if err := http.ListenAndServe(serveOn, nil); err != nil {
		log.Fatal(err)
	}
}
