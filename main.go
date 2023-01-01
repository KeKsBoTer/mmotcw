package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image/color"
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
	"unicode"

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
		if len(week.Maimais) == 0 {
			// if week folder is empty we remove it
			weeks = append(weeks[:i], weeks[i+1:]...)
		} else {
			weeks[i] = *week
		}
	}
	// sort weeks
	sort.Slice(weeks[:], func(i, j int) bool {
		return weeks[j].CW.Before(weeks[i].CW)
	})
	return weeks, nil
}

func index(template template.Template, source MaimaiSource, s *Subscriptions, users []string) http.HandlerFunc {

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
		w.Header().Add("Content-Type", "text/html")

		years := source.GetYears()

		err = template.Execute(w, struct {
			Weeks         []Week
			User          string
			PushPublicKey string
			Year          int
			Users         []string
			Years         []int
		}{
			Weeks:         maimais,
			User:          user,
			PushPublicKey: s.publicKey,
			Year:          year,
			Users:         users,
			Years:         years,
		})
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func userContent(template template.Template, source MaimaiSource, users []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := mux.Vars(r)["user"]
		year := getYear(r)
		weeks, err := GetMaimais(source, year)
		if err != nil {
			httpError(w, http.StatusInternalServerError)
			log.Error(err)
			return
		}
		found := false
		for _, u := range users {
			if u == user {
				found = true
			}
		}
		if !found {
			httpError(w, http.StatusNotFound)
			return
		}
		for w := range weeks {
			filtered := []UserMaimai{}
			for _, m := range weeks[w].Maimais {
				if strings.EqualFold(string(m.User), user) {
					filtered = append(filtered, m)
				}
			}
			weeks[w].Maimais = filtered
		}

		years := source.GetYears()

		err = template.Execute(w, struct {
			Weeks []Week
			User  string
			Years []int
		}{
			Weeks: weeks,
			User:  user,
			Years: years,
		})
		if err != nil {
			log.Error(err)
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

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

func createRouter(templates *template.Template, source MaimaiSource, sub *Subscriptions) *mux.Router {

	users, err := source.GetUsers()
	if err != nil {
		log.Fatalf("cannot load users: %s\n", err)
	}

	r := mux.NewRouter().StrictSlash(false)
	r.HandleFunc("/favicon.ico", faviconHandler)

	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// file server for maimais
	fsMaimais := http.FileServer(http.Dir(string(source)))
	r.PathPrefix("/mm/").Handler(http.StripPrefix("/mm/", fsMaimais))

	r.HandleFunc("/", index(*templates.Lookup("index.html"), source, sub, users))

	r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/js/sw.js")
	})

	r.HandleFunc("/upload", uploadHandler(source, sub))

	r.HandleFunc("/subscribe", subscribe(sub))

	r.HandleFunc("/{year:202[0-9]}/{user:[a-z]+}", userContent(*templates.Lookup("user.html"), source, users))

	r.HandleFunc("/{year:202[0-9]}", index(*templates.Lookup("index.html"), source, sub, users))

	r.HandleFunc("/{year:202[0-9]}/CW_{week:[0-9]+}", week(*templates.Lookup("week.html"), source))

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
		"add": func(a, b int) int {
			return a + b
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
			return fmt.Sprintf("%s %s", weekdays[w], t.Format("15:04"))
		},
		"capitalize": func(name string) string {
			s := []rune(name)
			if len(s) > 0 {
				s[0] = unicode.ToUpper(rune(name[0]))
			}
			return string(s)
		},
		"rgba": func(c color.Color) string {
			r, g, b, a := c.RGBA()
			return fmt.Sprintf("%d,%d,%d,%d", r/255, g/255, b/255, a/255)
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

	ImgCache.dir = string(source)

	if !skipCacheInit {
		go func() {
			err = FillCache()
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
