package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// GetImageFiles returns all images files located in the given folder
// image files end with jpg,jpeg,gif or png
func GetImageFiles(folder string) ([]os.FileInfo, error) {
	imgFiles, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	images := []os.FileInfo{}
	for _, img := range imgFiles {
		if !img.IsDir() {
			switch filepath.Ext(img.Name())[1:] {
			case
				"jpg",
				"jpeg",
				"gif",
				"png":
				images = append(images, img)
			}
		}
	}
	return images, nil
}

func getYear(r *http.Request) int {
	// change year if year is given
	if y, ok := mux.Vars(r)["year"]; ok {
		yy, _ := strconv.Atoi(y)
		return yy
	}
	return time.Now().Year()
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
