package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

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
