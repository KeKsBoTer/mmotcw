package main

import (
	"image"
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
func (m Maimai) Preview() Preview {
	p, err := ImgCache.GetPreview(m.Href())
	if err != nil {
		log.Error(err)
		p = &Preview{
			Size:  image.Pt(300, 300),
			Image: "",
		}
	}
	return *p
}
