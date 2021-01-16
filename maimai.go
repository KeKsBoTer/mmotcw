package main

import (
	"path/filepath"
	"time"
)

// Maimai stores information about an uploaded meme
type Maimai struct {
	File  string
	Time  time.Time
	Votes int
	CW    CW
}

// Href returns the relative url for the maimai
func (m Maimai) Href() string {
	return filepath.Join(m.CW.Path(), m.File)
}

// Preview returns the preview cached image
func (m Maimai) Preview() (CachedImage, error) {
	return ImgCache.GetImage(m.Href())
}
