package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// Week stores information about the maimais, votes etc. of a week
type Week struct {
	Maimais        []UserMaimai
	CW             CW
	Votes          Votes
	UserVotes      UserVotes
	CanVote        bool
	FinishedVoting bool
	// template file name
	Template *Template
}

// SortMaimais sorts the maimais by date
func (w *Week) SortMaimais() {
	sort.Slice(w.Maimais[:], func(i, j int) bool {
		a, b := w.Maimais[i], w.Maimais[j]
		return b.Before(a)
	})
}

// UserUploads counts the users upload in a week
func (w Week) UserUploads(user string) int {
	uploads := 0
	for _, m := range w.Maimais {
		if strings.ToLower(string(m.User)) == strings.ToLower(user) {
			uploads++
		}
	}
	return uploads
}

// ReadWeek reads all information for week from directory
func ReadWeek(directory string) (*Week, error) {
	source := MaimaiSource(filepath.Dir(filepath.Dir(directory)))
	cw, err := CWFromPath(directory)
	if err != nil {
		return nil, err
	}

	week, err := source.GetMaimaisForCW(*cw)
	if err != nil {
		return nil, err
	}

	uploadLock := CheckLock("upload", directory)
	voteLock := CheckLock("vote", directory)

	week.UserVotes = UserVotes{}
	week.Votes = Votes{}

	if CheckLock("upload", filepath.Join(string(source), cw.Path())) {
		userVotes, err := source.GetVoteResults(*cw)
		if err == nil {
			week.UserVotes = userVotes
			votes := userVotes.GetVotes()
			for i, v := range votes {
				votes[i].Path = filepath.Join(cw.Path(), v.FileName)
			}
			week.Votes = votes
		} else {
			log.Error(err)
		}

	}

	week.CanVote = uploadLock && !voteLock
	week.FinishedVoting = voteLock && uploadLock
	return week, nil
}
