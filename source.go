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
		Maimais: []UserMaimai{},
		CW:      cw,
	}
	for _, img := range imgFiles {

		if !strings.HasPrefix(img.Name(), "template.") {
			mm, err := NewUserMaimai(img.Name(), cw)
			if err != nil {
				log.Errorf("error in %d/%d/%s: %v", cw.Year, cw.Week, img.Name(), err)
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

// GetVoteResults reads voting results from directory
func (m MaimaiSource) GetVoteResults(cw CW) (UserVotes, error) {
	voteFilePath := filepath.Join(string(m), cw.Path(), "votes.txt")
	_, err := os.Stat(voteFilePath)
	if err == nil {
		votesFile, err := os.Open(voteFilePath)
		if err != nil {
			return nil, err
		}
		votes, err := ParseVotesFile(votesFile)
		if err != nil {
			return nil, err
		}

		return votes, nil
	}
	return nil, err
}
