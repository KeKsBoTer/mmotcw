package main

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MaimaiSource is a directory that containes all maimais
type MaimaiSource string

// GetMaimaisForCW reads all data from directory and returns a Week struct with the information
func (m MaimaiSource) GetMaimaisForCW(cw CW) (*Week, error) {
	imgFiles, err := GetImageFiles(filepath.Join(string(m), cw.Path()))
	if err != nil {
		return nil, err
	}
	week := Week{
		Maimais: []UserMaimai{},
		CW:      cw,
	}
	for _, img := range imgFiles {

		if !strings.HasPrefix(img.Name(), "template.") {
			mm, err := NewUserMaimai(img.Name(), img.ModTime(), cw)
			if err != nil {
				log.Errorf("error in %s/%s: %v", cw.Path(), img.Name(), err)
				continue
			}
			week.Maimais = append(week.Maimais, *mm)
		} else {
			week.Template = &Template{
				ImageType: strings.TrimPrefix(filepath.Ext(img.Name()), "."), // trim dot at start with TrimPrefix
				CW:        cw,
			}
		}
	}
	week.SortMaimais()
	return &week, nil
}

// Finds a calender week folders for a given year
func (m MaimaiSource) GetCWsOfYear(year int) ([]CW, error) {
	yearS := strconv.Itoa(year)
	yearPath := filepath.Join(string(m), yearS)
	files, err := os.ReadDir(yearPath)
	if err != nil {
		return nil, err
	}
	CWs := make([]CW, 0)
	for _, dirEntry := range files {
		if dirEntry.IsDir() && strings.HasPrefix(dirEntry.Name(), "CW_") {
			if cw, err := CWFromPath(path.Join(yearS, dirEntry.Name())); err == nil {
				CWs = append(CWs, *cw)
			} else {
				log.Error(err)
			}
		}
	}
	return CWs, nil
}

// returns all user listed in "users.txt"
func (m MaimaiSource) GetUsers() ([]string, error) {
	data, err := os.ReadFile(path.Join(string(m), "users.txt"))
	if err != nil {
		return nil, err
	}
	users := strings.Split(string(data), "\n")
	allUsers := make([]string, 0)
	for i := range users {
		name := strings.ToLower(strings.TrimSpace(users[i]))
		if len(name) > 0 {
			allUsers = append(allUsers, name)
		}
	}
	return allUsers, nil
}

// Finds all folders that have a year as name (regex: \d{4})
func (m MaimaiSource) GetYears() []int {
	yearFolders, err := filepath.Glob(path.Join(string(m), "[0-9][0-9][0-9][0-9]"))
	if err != nil {
		log.Error(err)
		now := [1]int{time.Now().Year()}
		return now[:]
	}
	years := make([]int, len(yearFolders))
	for i, y := range yearFolders {
		yi, _ := strconv.Atoi(path.Base(y))
		years[i] = yi
	}
	return years
}
