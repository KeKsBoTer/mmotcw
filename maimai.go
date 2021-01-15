package main

import (
	"image"
	"time"
)

// Maimai stores information about an uploaded meme
type Maimai struct {
	File      string
	Href      string
	Time      time.Time
	Votes     int
	ImageSize image.Point
	Preview   Base64String
}
