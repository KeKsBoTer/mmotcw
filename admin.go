package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

func adminPageHandler(template template.Template, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}

		canUpload := !CheckLock("upload", filepath.Join(string(source), cw.Path()))
		canVote := !CheckLock("vote", filepath.Join(string(source), cw.Path()))

		err := template.Execute(w, struct {
			CW        CW
			CanVote   bool
			CanUpload bool
		}{
			CW:        cw,
			CanVote:   canVote,
			CanUpload: canUpload,
		})
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func deleteLock(name string, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}
		cwDir := filepath.Join(string(source), cw.Path())

		if !CheckLock(name, cwDir) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		lockFile := filepath.Join(cwDir, fmt.Sprintf("%s.lock", name))
		if err := os.Remove(lockFile); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		referer := r.Header.Get("Referer")
		if referer == "" {
			referer = "/admin"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)
	}
}

func createLock(name string, source MaimaiSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}
		cwDir := filepath.Join(string(source), cw.Path())

		if CheckLock(name, cwDir) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		lockFile := filepath.Join(cwDir, fmt.Sprintf("%s.lock", name))
		if _, err := os.Create(lockFile); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		referer := r.Header.Get("Referer")
		if referer == "" {
			referer = "/admin"
		}
		http.Redirect(w, r, referer, http.StatusSeeOther)
	}
}

func adminRouter(template template.Template, source MaimaiSource) *mux.Router {
	r := mux.NewRouter()
	r.Handle("/admin", adminPageHandler(template, source))
	// upload lock
	r.Path("/admin/upload/delete").
		HandlerFunc(deleteLock("upload", source))
	r.Path("/admin/upload/create").
		HandlerFunc(createLock("upload", source))
	// vote lock
	r.Path("/admin/vote/delete").
		HandlerFunc(deleteLock("vote", source))
	r.Path("/admin/vote/create").
		HandlerFunc(createLock("vote", source))
	return r
}
