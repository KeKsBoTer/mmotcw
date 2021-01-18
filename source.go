package main

import (
	"os"
	"path/filepath"
	"strings"
)

// TODO maybe only expose interface for file reading

// MaimaiSource is a directory that containes all maimais
type MaimaiSource string

// GetMaimaisForCW reads all data from directory and returns a Week struct with the information
func (m MaimaiSource) GetMaimaisForCW(cw CW) (*Week, error) {
	imgFiles, err := GetImageFiles(filepath.Join(string(m), cw.Path()))
	if err != nil {
		return nil, err
	}
	week := Week{
		Maimais: []Maimai{},
		CW:      cw,
	}
	for _, img := range imgFiles {
		mm := Maimai{
			File: img.Name(),
			Time: img.ModTime(),
			CW:   cw,
		}
		if !strings.HasPrefix(img.Name(), "template.") {
			week.Maimais = append(week.Maimais, mm)
		} else {
			week.Template = &mm
		}
	}
	week.SortMaimais()
	return &week, nil
}

// GetVoteResults reads voting results from directory
func (m MaimaiSource) GetVoteResults(cw CW) ([]Vote, error) {
	voteFilePath := filepath.Join(string(m), cw.Path(), "votes.txt")
	if _, err := os.Stat(voteFilePath); err == nil {
		votesFile, err := os.Open(voteFilePath)
		if err != nil {
			return nil, err
		}
		votes, err := parseVotesFile(votesFile)
		if err != nil {
			return nil, err
		}
		for i, v := range votes {
			votes[i].Path = filepath.Join(string(m), cw.Path(), v.FileName)
		}
		return votes, nil
	}
	return nil, nil
}