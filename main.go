package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Maimai struct {
	File  string
	Href  string
	Time  time.Time
	Votes int
}

type Week struct {
	Maimais       []Maimai
	KW            int
	IsCurrentWeek bool
	Votes         []Vote
	CanVote       bool
}

type Vote = struct {
	FileName string
	Votes    int
	Path     string
}

func voteCount(i int) int {
	return int(math.Sqrt(float64(i)) * 1.15)
}



func checkLock(name string, weekFolder string) bool {
	fileName := fmt.Sprintf("%s.lock", name)
	filePath := filepath.Join(weekFolder, fileName)
	_, err := os.Stat(filePath)
	return err == nil
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
		uploadLock := checkLock("upload", w)
		cw, _ := strconv.Atoi(filepath.Base(w)[3:])
		if checkLock("vote", w) && uploadLock {
			votes, err := getVoteResults(w, cw)
			if err == nil {
				week.Votes = votes
			} else {
				log.Println(err)
			}
		}
		week.CanVote = uploadLock

		weeks[i] = *week
	}
	sort.Slice(weeks[:], func(i, j int) bool {
		return weeks[i].KW > weeks[j].KW
	})
	return weeks, nil
}

func getMaiMaiPerCW(pathPrefix string, w string) (*Week, error) {
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
					File: img.Name(),
					Href: filepath.Join(pathPrefix, filepath.Base(w), img.Name()),
					Time: img.ModTime()})
				break
			}
		}
	}
	sort.Slice(week.Maimais[:], func(i, j int) bool {
		return week.Maimais[i].Time.After(week.Maimais[j].Time)
	})
	return &week, nil
}

func index(template template.Template, directory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Geheimwort bitte"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 page not found"))
			return
		}
		maimais, err := getMaimais(directory)
		if err != nil {
			log.Fatalln(err)
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

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

func sortVotes(votes map[string]int) []Vote {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range votes {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	ranked := make([]string, len(votes))
	for i, kv := range ss {
		ranked[i] = kv.Key
	}

	votesList := make([]Vote, len(ranked))
	for i, key := range ranked {
		votesList[i] = Vote{
			FileName: key,
			Votes:    votes[key],
		}
	}
	return votesList
}

func parseVotesFile(file io.Reader) ([]Vote, error) {
	reader := csv.NewReader(file)
	reader.Comma = ':'
	reader.FieldsPerRecord = -1
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	votes := map[string]int{}
	for _, line := range lines {
		for _, maimai := range line[1:] {
			if count, ok := votes[maimai]; ok {
				votes[maimai] = count + 1
			} else {
				votes[maimai] = 1
			}
		}
	}

	return sortVotes(votes), nil
}

func getVoteResults(weekDir string, week int) ([]Vote, error) {
	voteFilePath := filepath.Join(weekDir, "votes.txt")
	if _, err := os.Stat(voteFilePath); err == nil {
		votesFile, err := os.Open(voteFilePath)
		if err != nil {
			return nil, err
		}
		votes, err := parseVotesFile(votesFile)
		if err != nil {
			return nil, err
		}
		for i, v := range votes {
			weekString := fmt.Sprintf("CW_%d", week)
			votes[i].Path = filepath.Join("mm", weekString, v.FileName)
		}
		return votes, nil
	}
	return nil, nil
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
		week := mux.Vars(r)["week"]
		cwPath := filepath.Join(directory, "CW_"+week)
		maimais, err := getMaiMaiPerCW("mm", cwPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - server ist kaputt"))
		}

		err = template.Execute(w, struct {
			Maimais Week
			Week    string
		}{
			Maimais: *maimais,
			Week:    week,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func main() {
	var directory = flag.String("dir", ".", "the maimai directory")
	var port = flag.Int("port", 8080, "port to run on")
	flag.Parse()

	funcMap := template.FuncMap{
		"numvotes": func(maimais []Maimai) []int {
			v := voteCount(len(maimais))
			votes := make([]int, v)
			for i := range votes {
				votes[i] = i
			}
			return votes
		},
		"add": func(a , b int) int {
			return a+b
		},
	}

	tmpl := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html"))
	userTmpl := template.Must(template.New("user.html").Funcs(funcMap).ParseFiles("templates/user.html"))
	weekTmpl := template.Must(template.New("week.html").Funcs(funcMap).ParseFiles("templates/week.html"))

	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", faviconHandler)

	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// file server for maimais
	fsMaimais := http.FileServer(http.Dir(*directory))
	r.PathPrefix("/mm/").Handler(http.StripPrefix("/mm/", fsMaimais))

	r.HandleFunc("/", index(*tmpl, *directory))

	r.HandleFunc("/{user:[a-z]+}", userContent(*userTmpl, *directory))

	r.HandleFunc("/CW_{week:[0-9]+}", week(*weekTmpl, *directory))

	http.Handle("/", r)

	if err := http.ListenAndServe(":"+strconv.Itoa(*port), nil); err != nil {
		log.Fatalln(err)
	}
}
