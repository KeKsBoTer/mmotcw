package main

import (
	"encoding/csv"
	"io"
	"math"
	"sort"
)

// Vote represents the votes for a file
type Vote struct {
	FileName string
	Votes    int
	Path     string
}

// voteCount calculates the number of votes per user
func voteCount(i int) int {
	return int(math.Sqrt(float64(i)) * 1.15)
}

func sortVotes(votes map[string]int) []Vote {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range votes {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	ranked := make([]string, len(votes))
	for i, kv := range ss {
		ranked[i] = kv.Key
	}

	votesList := make([]Vote, len(ranked))
	for i, key := range ranked {
		votesList[i] = Vote{
			FileName: key,
			Votes:    votes[key],
		}
	}
	return votesList
}

func parseVotesFile(file io.Reader) ([]Vote, error) {
	reader := csv.NewReader(file)
	reader.Comma = ':'
	reader.FieldsPerRecord = -1
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	votes := map[string]int{}
	for _, line := range lines {
		for _, maimai := range line[1:] {
			if count, ok := votes[maimai]; ok {
				votes[maimai] = count + 1
			} else {
				votes[maimai] = 1
			}
		}
	}

	return sortVotes(votes), nil
}