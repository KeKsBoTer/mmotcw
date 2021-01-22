package main

import (
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func checkYearFolder(cw CW, path string) (string, error) {

	folderInfo, err := os.Stat(cw.Path())
	if os.IsNotExist(err) {
		err := os.Mkdir(path+strconv.Itoa(cw.Year), 0755)
		if err != nil {
			log.Error(err)
			return "", err
		}

		return path + strconv.Itoa(cw.Year) + "/", nil

	}
	log.Info(folderInfo.Name())
	return path + strconv.Itoa(cw.Year) + "/", nil

}

func checkCWFolder(cw CW, path string) (string, error) {

	folderInfo, err := os.Stat(cw.Path())
	if os.IsNotExist(err) {
		err := os.Mkdir(cw.Path(), 0755)
		if err != nil {
			log.Error(err)
			return "", err
		}
		return cw.Path() + "/", nil
	}
	log.Info(folderInfo.Name())
	return cw.Path() + "/", nil

}

func detectType(f multipart.File) (string, error) {
	buffer := make([]byte, 512)
	_, err := f.Read(buffer)
	//_, err := f.Read(buffer)
	if err != nil {
		log.Info(err.Error())
		return "Could not read to Buffer", err
	}

	f.Seek(0, 0)

	return http.DetectContentType(buffer), nil
}

func countFiles(cw CW) (int, error) {
	files, err := ioutil.ReadDir(cw.Path() + "/")
	if err != nil {
		log.Error(err)
		return 0, err
	}

	return len(files), nil
}

func countFilesUser(cw CW, name string) (int, error) {
	counter := 0
	files, err := ioutil.ReadDir(cw.Path())
	if err != nil {
		log.Error(err)
		return 0, err
	}
	for _, f := range files {
		if strings.Contains(f.Name(), name) {
			counter++
		}
	}

	return counter, nil
}
