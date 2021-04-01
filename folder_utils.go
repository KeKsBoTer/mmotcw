package main

import (
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func checkCWFolder(cw CW, path string) (string, error) {

	folderInfo, err := os.Stat(filepath.Join(path, cw.Path()))
	if os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Join(path, cw.Path()), 0755)
		if err != nil {
			log.Error(err)
			return "", err
		}
		return filepath.Join(path, cw.Path(), "/"), nil
	}
	log.Info(folderInfo.Name())
	return filepath.Join(path, cw.Path(), "/"), nil

}

func detectType(f multipart.File) (string, error) {
	buffer := make([]byte, 512)
	_, err := f.Read(buffer)
	if err != nil {
		log.Info(err.Error())
		return "Could not read to Buffer", err
	}

	f.Seek(0, 0)

	return http.DetectContentType(buffer), nil
}
