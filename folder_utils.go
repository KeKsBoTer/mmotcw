package main

import (
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func checkYearFolder(year string, path string) string {

	folderInfo, err := os.Stat(path + year)
	if os.IsNotExist(err) {
		err := os.Mkdir(path+year, 0755)
		if err != nil {
			log.Fatal(err)
			return "false"
		}

		return path + year + "/"

	}
	fmt.Printf(folderInfo.Name())
	return path + year + "/"

}

func checkCWFolder(cw string, path string) string {

	folderInfo, err := os.Stat(path + cw)
	if os.IsNotExist(err) {
		err := os.Mkdir(path+cw, 0755)
		if err != nil {
			log.Fatal(err)
			return "false"
		}
		return path + cw + "/"
	}
	fmt.Printf(folderInfo.Name())
	return path + cw + "/"

}

func detectType(f multipart.File) string {
	buffer := make([]byte, 512)
	_, err := f.Read(buffer)
	//_, err := f.Read(buffer)
	if err != nil {
		fmt.Printf(err.Error())
		return "Could not read to Buffer"
	}

	f.Seek(0, 0)

	return http.DetectContentType(buffer)
}

func countFiles(path string) int {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	return len(files)
}

func countFilesUser(path string, name string) int {
	counter := 0
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(f.Name(), name) {
			counter++
		}
	}

	return counter
}
