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

// Votes contains a list of votes
type Votes []Vote

// voteCount calculates the number of votes per user
func voteCount(i int) int {
	return int(math.Sqrt(float64(i)) * 1.15)
}

func sortVotes(votes map[string]int) Votes {
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

	votesList := make(Votes, len(ranked))
	for i, key := range ranked {
		votesList[i] = Vote{
			FileName: key,
			Votes:    votes[key],
		}
	}
	return votesList
}

// TODO merge Vote and UserVote

// UserVotes is a list of votes
type UserVotes map[string][]string

// ParseVotesFile reads a votes file
func ParseVotesFile(file io.Reader) (UserVotes, error) {
	reader := csv.NewReader(file)
	reader.Comma = ':'
	reader.FieldsPerRecord = -1
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	votes := UserVotes{}
	for _, line := range lines {
		votes[line[0]] = line[1:]
	}

	return votes, nil
}

// WriteToFile saves user votes in a given file as csv
func (votes UserVotes) WriteToFile(file io.Writer) error {
	writer := csv.NewWriter(file)
	writer.Comma = ':'
	writer.UseCRLF = false

	for user, v := range votes {
		//TODO store user
		line := make([]string, len(v)+1)
		line[0] = user
		copy(line[1:], v)
		err := writer.Write(line)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

// GetVotes returns the votes for each maimai in descending order (sorted by number of votes)
func (votes UserVotes) GetVotes() Votes {
	v := map[string]int{}
	for _, userVotes := range votes {
		for _, maimai := range userVotes {
			if count, ok := v[maimai]; ok {
				v[maimai] = count + 1
			} else {
				v[maimai] = 1
			}
		}
	}

	return sortVotes(v)
}

// SetVotes sets a users votes
func (votes *UserVotes) SetVotes(user string, newVotes []string) {
	(*votes)[user] = newVotes
}
