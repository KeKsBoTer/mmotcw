package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
)

// CW represents a calender week and its year
type CW struct {
	Week int
	Year int
}

// CWFromPath parses a path of style '../YYYY/CW_WW' (YYYY = year, WW = week)
func CWFromPath(path string) (*CW, error) {
	kw := filepath.Base(path)
	if matches, err := regexp.MatchString("CW_\\d{2}", kw); !matches || err != nil {
		return nil, errors.New("calender week is not of expected format 'CW_WW")
	}
	cw, _ := strconv.Atoi(kw[3:])
	yearStr := filepath.Base(filepath.Dir(path))
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, errors.New("year is not a digit")
	}
	return &CW{Year: year, Week: cw}, nil

}

// ImagePath returns relative path for image
func (c CW) ImagePath(imgName string) string {
	year := strconv.Itoa(c.Year)
	cw := fmt.Sprintf("CW_%02d", c.Week)
	return filepath.Join(year, cw, imgName)
}

// Before checks if one calender week is before the other in time
func (c CW) Before(c2 CW) bool {
	return c.Year < c2.Year || (c.Year == c2.Year && c.Week < c2.Week)
}
