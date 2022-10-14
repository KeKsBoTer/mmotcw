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
	CanVote        bool
	FinishedVoting bool
	// template file name
	Template *Template
}

// SortMaimais sorts the maimais by date
func (w *Week) SortMaimais() {
	sort.SliceStable(w.Maimais[:], func(i, j int) bool {
		a, b := w.Maimais[i], w.Maimais[j]
		return b.Before(a)
	})
}

// UserUploads counts the users upload in a week
func (w Week) UserUploads(user string) int {
	uploads := 0
	for _, m := range w.Maimais {
		if strings.EqualFold(string(m.User), user) {
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
	return week, nil
}
