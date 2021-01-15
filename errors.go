package main

import "net/http"

func error505(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - server ist kaputt"))
}
func error404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - guck wo anders hin"))
}

func httpError(w http.ResponseWriter, code int) {
	switch code {
	case 505:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - server ist kaputt"))
		return
	case 404:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 - guck wo anders hin"))
	default:
		w.WriteHeader(500)
		w.Write([]byte("Schiefer kanns nicht mehr laufen"))
	}
}
