package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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

func uploadHandler(source MaimaiSource, s *Subscriptions) http.HandlerFunc {
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
		ext := "png"
		switch mimeType {
		case "image/gif":
			ext = "gif"
		case "image/png":
			ext = "png"
		case "image/jpeg":
			ext = "jpg"
		default:
			fmt.Fprintf(w, "Deine Datei wollen wir hier nicht: %s %s", mimeType, handler.Filename)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		year, week := time.Now().ISOWeek()
		cw := CW{Year: year, Week: week}
		if CheckLock("upload", filepath.Join(string(source), cw.Path())) {
			fmt.Fprint(w, `
				<h1>Upload nicht mehr möglich!</h1>
				<p>Sehr geehrte[r] Pfostierer:in!</p>
				<p>Das Hochladen ist Freitags bis 16:45 Uhr möglich.</p>
			`)
			return
		}

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
		weekData, err := ReadWeek(folderCW)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
		}

		cFiles := len(weekData.Maimais) + 1
		cFilesUser := weekData.UserUploads(user)
		name := fmt.Sprintf("%d_%s_%d.%s", cFiles, user, cFilesUser, ext)

		osFile, err := os.Create(filepath.Join(folderCW, name))

		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}
		defer osFile.Close()

		_, err = io.Copy(osFile, file)
		if err != nil {
			log.Error(err)
			httpError(w, http.StatusInternalServerError)
			return
		}

		s.Send(fmt.Sprintf("%s has ein Maimai pfostiert", user))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
