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

func index(template template.Template, source MaimaiSource) http.HandlerFunc {

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

func vote(source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, http.StatusMethodNotAllowed)
			return
		}
		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}
		if !CheckLock("upload", filepath.Join(string(source), cw.Path())) {
			fmt.Fprint(w, `
				<h1>Abstimmung noch nicht möglich!</h1>
				<p>Sehr geehrte[r] Pfostierer:in!</p>
				<p>Das Abstimmen ist Freitags ab 16:45 Uhr möglich.</p>
			`)
			return
		}
		if CheckLock("vote", filepath.Join(string(source), cw.Path())) {
			fmt.Fprint(w, `
				<h1>Abstimmung beendet!</h1>
				<p>Sehr geehrte[r] Pfostierer:in!</p>
				<p>Die Abstimmung ist bereits beendet.</p>
			`)
			return
		}

		user, _, _ := r.BasicAuth()

		err := r.ParseMultipartForm(0)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}
		data := r.MultipartForm.Value
		mm, _ := ReadWeek(filepath.Join(string(source), cw.Path()))
		if len(data) != voteCount(len(mm.Maimais)) {
			httpError(w, http.StatusBadRequest)
			return
		}
		votes := make([]string, len(data))
		for i := 0; i < len(data); i++ {
			votes[i] = data[strconv.Itoa(i)][0]
		}
		mm.UserVotes.SetVotes(user, votes)

		votesPath := filepath.Join(string(source), cw.Path(), "votes.txt")
		file, err := os.Create(votesPath)
		if err != nil {
			httpError(w, http.StatusInternalServerError)
			log.Error(err)
			return
		}
		mm.UserVotes.WriteToFile(file)
		file.Close()

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func uploadHandler(source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("fileToUpload")
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusBadRequest)
			return
		}
		defer file.Close()

		mimeType, err := detectType(file)
		if err != nil {
			mimeType = ""
			err = nil
		}
		ext := ".png"
		switch mimeType {
		case "image/gif":
			ext = ".gif"
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpg"
		default:
			fmt.Fprintf(w, "Deine Datei wollen wir hier nicht: "+mimeType+" "+handler.Filename)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}
		path := string(source)
		log.Info(cw.Path())
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		folderCW, err := checkCWFolder(cw, path)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		user, _, _ := r.BasicAuth()

		cFiles, err := countFiles(cw, path)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}
		cFilesUser, err := countFilesUser(cw, user, path)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}
		name := fmt.Sprintf("%d_%s_%d", cFiles, user, cFilesUser)

		osFile, err := os.OpenFile(filepath.Join(folderCW, name+ext), os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}
		defer osFile.Close()
		io.Copy(osFile, file)

		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	r.HandleFunc("/vote", vote(source))

	r.HandleFunc("/upload", uploadHandler(source))

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

// loadTemplates reads all .html files as templates from given directory
func loadTemplates(dir string) *template.Template {

	funcMap := template.FuncMap{
		"numVotes": func(maimais []Maimai) []int {
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
		"toJson": func(i interface{}) string {
			b, err := json.Marshal(i)
			if err != nil {
				log.Error(err)
				return ""
			}
			return base64.RawStdEncoding.EncodeToString(b)
		},
	}

	return template.Must(template.New("templates").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html")))

}

func main() {
	// enable debug
	if os.Getenv("DEBUG") == "true" {
		log = log.WithDebug()
	}
	miamaiDir, port := readFlags()

	ImgCache.dir = miamaiDir

	templates := loadTemplates("./templates")

	source := MaimaiSource(miamaiDir)
	router := createRouter(templates, source)
	http.Handle("/", router)

	err := InitCache(source)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("starting webserver on http://localhost:%d", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}
