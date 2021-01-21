package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

// ImgCache is the global cache for all images
var ImgCache = PreviewCache{
	dir:   "",
	cache: sync.Map{},
}

// Base64String is a string that holds base64 encoded content
type Base64String string

// CachedImage is a low dimensional cached version of an image
type CachedImage struct {
	Size  image.Point
	Image Base64String
}

// PreviewCache is a key-value map with cached preview images
type PreviewCache struct {
	dir   string
	cache sync.Map
}

// GetImage returns Base54 encoded image of path
// returns empty image if something goes wrong
func (c *PreviewCache) GetImage(imgPath string) (CachedImage, error) {
	if cache, ok := c.cache.Load(imgPath); ok {
		return cache.(CachedImage), nil
	}
	err := c.cacheImage(imgPath)
	if err != nil {
		return CachedImage{Image: ""}, err
	}
	img, _ := c.cache.Load(imgPath)
	return img.(CachedImage), nil
}

func (c *PreviewCache) cacheImage(imgPath string) error {
	filePath := filepath.Join(c.dir, imgPath)
	if imgFile, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm); err == nil {
		img, _, err := image.Decode(imgFile)
		if err != nil {
			return err
		}
		ratio := float32(img.Bounds().Size().Y) / float32(img.Bounds().Size().X)

		cachedImage := CachedImage{
			Size:  image.Point{330, 330},
			Image: "",
		}

		cachedImage.Size.Y = int(ratio * float32(cachedImage.Size.X))
		// create smal image preview
		smallImage := resize.Resize(20, uint(ratio*20), img, resize.Lanczos3)
		buffer := bytes.NewBuffer([]byte{})
		if err := jpeg.Encode(buffer, smallImage, nil); err != nil {
			return err
		}
		cachedImage.Image = Base64String(base64.RawStdEncoding.EncodeToString(buffer.Bytes()))
		c.cache.Store(imgPath, cachedImage)
		log.Debugf("added image '%s' to cache", imgPath)
	} else {
		return err
	}
	return nil
}

// InitCache loads images for current year into cache
func InitCache(source MaimaiSource) error {
	year, _ := time.Now().ISOWeek()
	weeks, err := GetMaimais(source, year)
	if err != nil {
		return err
	}

	// load an image an send info through channel once done
	load := func(m Maimai, c chan int) {
		m.Preview()
		c <- 1
	}

	var c chan int = make(chan int)

	log.Infof("loading image preview cache for year %d...", year)
	i := 0
	for _, w := range weeks {
		for _, m := range w.Maimais {
			go load(m, c)
			i++
		}
	}
	// wait for all image loading threads to return
	for j := 0; j < i; j++ {
		<-c
	}
	log.Info("Done")
	return nil
}
