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

// Extracts current year from request
// checks for the mux var "year" and tries to convert it to an int
// If the param is not present or not an integer the current year is returned
func getYear(r *http.Request) int {
	// change year if year is given
	if y, ok := mux.Vars(r)["year"]; ok {
		if yy, err := strconv.Atoi(y); err != nil {
			year, _ := time.Now().ISOWeek()
			return year
		} else {
			return yy
		}
	}
	year, _ := time.Now().ISOWeek()
	return year
}
