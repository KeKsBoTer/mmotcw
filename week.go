package main

import (
	"path/filepath"
	"sort"
)

// Week stores information about the maimais, votes etc. of a week
type Week struct {
	Maimais []Maimai
	CW      CW
	Votes   []Vote
	CanVote bool
	// template file name
	Template *Maimai
}

// SortMaimais sorts the maimais by date
func (w *Week) SortMaimais() {
	sort.Slice(w.Maimais[:], func(i, j int) bool {
		return w.Maimais[i].Time.After(w.Maimais[j].Time)
	})
}

// ReadWeek reads all imformation for week from directory
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

	if voteLock && uploadLock {
		votes, err := source.GetVoteResults(*cw)
		if err == nil {
			week.Votes = votes
		} else {
			log.Error(err)
		}
	}
	week.CanVote = uploadLock && !voteLock
	return week, nil
}
