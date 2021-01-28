package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chai2010/webp"
)

// ImgCache is the global cache for all images
var ImgCache ImageCache

// ImageCache is a key-value map with cached preview images
type ImageCache struct {
	cacheDir string
	source   MaimaiSource
}

// Base64String is a string that holds base64 encoded content
type Base64String string

// Preview is a small image (around 20x20 pixel)
type Preview struct {
	Size  image.Point
	Image Base64String
}

// GetImage loads the image from cache
// caches it first, if needed
func (c *ImageCache) GetImage(imgPath string) ([]byte, error) {
	fp := c.getCachePath(imgPath, false)

	f, err := os.Open(fp)
	if _, ok := err.(*os.PathError); ok {
		// image is not cached yet
		data, err := c.cacheImage(imgPath)
		if err != nil {
			return nil, err
		}
		return data, nil
	} else if err != nil {
		// something unforeseen went wrong
		return nil, err
	}

	// cache does allready exist
	return ioutil.ReadAll(f)
}

func (c *ImageCache) getCachePath(path string, preview bool) string {
	imgPath := path
	if preview {
		imgPath += ".preview"
	}

	// create hash
	h := sha1.New()
	h.Write([]byte(imgPath))
	bs := h.Sum(nil)
	return filepath.Join(string(c.cacheDir), "mmotcw", fmt.Sprintf("%x", bs))
}

// GetPreview loads the preview image from cache
// caches it first, if needed
func (c *ImageCache) GetPreview(imgPath string) (*Preview, error) {
	fp := c.getCachePath(imgPath, true)

	f, err := os.Open(fp)
	if _, ok := err.(*os.PathError); ok {
		// image is not cached yet
		p, err := c.cachePreview(imgPath)
		if err != nil {
			return nil, err
		}
		return p, nil
	} else if err != nil {
		// something unforeseen went wrong
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), " ")
	if len(parts) != 3 {
		// cache is corrupted
		return c.cachePreview(fp)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		// cache is corrupted
		return c.cachePreview(fp)
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		// cache is corrupted
		return c.cachePreview(fp)
	}
	preview := Base64String(parts[2])
	return &Preview{
		Size:  image.Pt(300, int(300/float32(width)*float32(height))),
		Image: preview,
	}, nil
}

// caches image and returns the content
func (c *ImageCache) cacheImage(imgPath string) ([]byte, error) {
	f, err := os.Open(c.source.GetPath(imgPath))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	err = webp.Encode(&buffer, img, &webp.Options{Lossless: false, Quality: 90, Exact: true})
	if err != nil {
		return nil, err
	}
	// save cache file
	fp := c.getCachePath(imgPath, false)
	fc, err := os.Create(fp)
	if err != nil {
		return nil, err
	}
	data := buffer.Bytes()
	fc.Write(data)
	if err != nil {
		return nil, err
	}
	return data, err
}

func (c *ImageCache) cachePreview(imgPath string) (*Preview, error) {
	imgData, err := c.cacheImage(imgPath)
	if err != nil {
		return nil, err
	}
	width, height, _, err := webp.GetInfo(imgData)
	if err != nil {
		return nil, err
	}

	// resize to 20 pixel height
	ratio := float32(height) / float32(width)
	previewWidth := 5
	previewHeight := int(ratio * float32(previewWidth))

	var data []byte

	imgRGB, err := webp.DecodeRGBToSize(imgData, previewWidth, previewHeight)
	if err != nil {
		return nil, err
	}
	data, err = webp.EncodeRGB(imgRGB, float32(1))

	if err != nil {
		return nil, err
	}

	fp := c.getCachePath(imgPath, true)
	f, err := os.Create(fp)
	if err != nil {
		return nil, err
	}
	preview := base64.RawStdEncoding.EncodeToString(data)
	fmt.Fprintf(f, "%d %d %s", width, height, preview)
	return &Preview{
		Size:  image.Pt(300, int(300/float32(width)*float32(height))),
		Image: Base64String(preview),
	}, nil
}

// InitCache loads images for current year into cache
func InitCache(source MaimaiSource) error {

	err := os.MkdirAll(filepath.Join(ImgCache.cacheDir, "mmotcw"), os.ModeDir)
	if err != nil {
		return err
	}

	var c chan int = make(chan int)
	count := 0
	year := time.Now().Year()
	for i := 0; i < 3; i++ {
		log.Infof("loading image preview cache for year %d...", year)
		year--
		weeks, err := GetMaimais(source, year)
		if err != nil {
			return err
		}

		// load an image an send info through channel once done
		load := func(m Maimai, c chan int) {
			m.Preview()
			c <- 1
		}

		for _, w := range weeks {
			for _, m := range w.Maimais {
				go load(m, c)
				count++
			}
		}
	}
	// wait for all image loading threads to return
	for j := 0; j < count; j++ {
		<-c
	}
	log.Info("Done")
	return nil
}
