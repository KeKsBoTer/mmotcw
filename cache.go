package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
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

// GetImage returns Base64 encoded image of path
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
	imgFile, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer imgFile.Close()
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
	res := uint(8)
	// create small image preview
	smallImage := resize.Thumbnail(res, res, img, resize.Lanczos3)
	buffer := bytes.NewBuffer([]byte{})
	if err := jpeg.Encode(buffer, smallImage, nil); err != nil {
		return err
	}
	cachedImage.Image = Base64String(base64.RawStdEncoding.EncodeToString(buffer.Bytes()))
	c.cache.Store(imgPath, cachedImage)
	log.Debugf("added image '%s' to cache", imgPath)
	return nil
}

// InitCache loads images for current year and last three years into cache
func FillCache() error {

	year, _ := time.Now().ISOWeek()

	worker := func(jobs <-chan Maimai, wg *sync.WaitGroup) {
		defer wg.Done()
		for m := range jobs {
			_, err := m.Preview()
			if err != nil {
				log.Warnf("cannot precache image %e ", err)
			}
		}
	}

	var wg sync.WaitGroup
	var jobs chan Maimai = make(chan Maimai)

	for w := 1; w <= runtime.NumCPU(); w++ {
		wg.Add(1)
		go worker(jobs, &wg)
	}

	for i := 0; i < 3; i++ {
		weeks, err := GetMaimais(MaimaiSource(ImgCache.dir), year-i)
		if err != nil {
			return err
		}
		if len(weeks) == 0 {
			// nothing to cache
			continue
		}

		log.Infof("loading image preview cache for year %d...", year-i)
		for _, w := range weeks {
			for _, m := range w.Maimais {
				jobs <- m
			}
		}
	}
	close(jobs)
	wg.Wait()
	log.Info("Done")
	return nil
}
