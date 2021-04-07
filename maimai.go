package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Maimai is an interface for all uploaded Maimais (user uploads and templates)
type Maimai interface {
	Href() string
	Preview() (CachedImage, error)
	FileName() string
}

// UserName is the name of a user ;)
type UserName string

// UserMaimai stores information about an uploaded meme
type UserMaimai struct {
	// the file name of the maimai
	User UserName

	// UploadTime is the upload date
	UploadTime time.Time

	// number in current week (unique)
	Counter int

	// number for users upload
	UserCounter int

	// image type e.g. jpg, jpeg, png, gif
	ImageType string

	// calender week the maimai belongs to
	CW CW
}

// NewUserMaimai creates a Maimai object from a filename
func NewUserMaimai(fileName string, uploadTime time.Time, cw CW) (*UserMaimai, error) {
	err := fmt.Errorf("%s is not of expected file name format for a maimai", fileName)
	parts := strings.Split(fileName, ".")
	if len(parts) != 2 {
		return nil, err
	}
	name := parts[0]
	imageType := parts[1]

	parts = strings.Split(name, "_")
	var counter, userCounter int
	var userName string
	if len(parts) != 3 {
		return nil, err
	}

	counter, err = strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	userName = parts[1]

	userCounter, err = strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	return &UserMaimai{
		User:        UserName(userName),
		UploadTime:  uploadTime,
		Counter:     counter,
		UserCounter: userCounter,
		ImageType:   imageType,
		CW:          cw,
	}, nil
}

// Href returns the relative url for the maimai
func (m UserMaimai) Href() string {
	return filepath.Join(m.CW.Path(), m.FileName())
}

// FileName returns the maimais filename
// e.g. 12_hans_1.gif
func (m UserMaimai) FileName() string {
	return fmt.Sprintf("%d_%s_%d.%s", m.Counter, m.User, m.UserCounter, m.ImageType)
}

// Preview returns the preview cached image
func (m UserMaimai) Preview() (CachedImage, error) {
	return ImgCache.GetImage(m.Href())
}

// Before returns true if counter is smaller than the one it is compared to
func (m UserMaimai) Before(a UserMaimai) bool {
	return m.Counter < a.Counter
}

// Template image for a week
type Template struct {
	CW        CW
	ImageType string
}

// Href returns the relative url for the maimai
func (m Template) Href() string {
	return filepath.Join(m.CW.Path(), m.FileName())
}

// FileName returns the maimais filename
// e.g. 12_hans_1.gif
func (m Template) FileName() string {
	return fmt.Sprintf("template.%s", m.ImageType)
}

// Preview returns the preview cached image
func (m Template) Preview() (CachedImage, error) {
	return ImgCache.GetImage(m.Href())
}
