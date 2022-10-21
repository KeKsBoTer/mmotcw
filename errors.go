package main

import (
	"fmt"
	"net/http"
)

func httpError(w http.ResponseWriter, code int) {
	switch code {
	case http.StatusInternalServerError:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "500 - server ist kaputt")
		return
	case http.StatusNotFound:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "404 - guck wo anders hin")
	case http.StatusMethodNotAllowed:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "hier wird nur gePOSTed!")
	case http.StatusUnauthorized:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "ich kenn dich nicht!")
	case http.StatusBadRequest:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "so kann ich nicht arbeiten")
	default:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Schiefer kanns nicht mehr laufen")
	}
}
