package main

import (
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// copied from http module since it is not public :(
func toHTTPError(err error) (msg string, httpStatus int) {
	if os.IsNotExist(err) {
		return "404 page not found", http.StatusNotFound
	}
	if os.IsPermission(err) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}

// ImageServer is the same as http.FileServer
func ImageServer(fs http.FileSystem) http.Handler {
	return &imageHandler{fs}
}

type imageHandler struct {
	fs http.FileSystem
}

type imageDecoder func(r io.Reader) (image.Image, error)

func (h *imageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ext := filepath.Ext(r.URL.Path)
	decoders := map[string]string{
		".jpeg": "",
		".jpg":  "",
		".png":  "",
		//".gif":  "", //TODO support animated GIFs
	}
	if _, ok := decoders[ext]; ok {
		data, err := ImgCache.GetImage(r.URL.Path)
		if err != nil {
			log.Error(err)
			http.FileServer(h.fs).ServeHTTP(w, r)
			return
		}
		w.Write(data)
	} else {
		http.FileServer(h.fs).ServeHTTP(w, r)
	}
}
