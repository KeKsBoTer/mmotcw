package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

// ImgCache is the global cache for all images
var ImgCache = PreviewCache{
	dir:   "",
	cache: map[string]CachedImage{},
}

// Base64String is a string that bolds base64 encoded content
type Base64String string

// CachedImage is a low dimensional chached version of an image
type CachedImage struct {
	Size  image.Point
	Image Base64String
}

// PreviewCache is a key-value map with cached preview images
type PreviewCache struct {
	dir   string
	cache map[string]CachedImage
}

// GetImage returns Base54 encoded image of path
// returns empty image if something goes wrong
func (c PreviewCache) GetImage(imgPath string) (CachedImage, error) {
	if cache, ok := c.cache[imgPath]; ok {
		return cache, nil
	}
	err := c.cacheImage(imgPath)
	if err != nil {
		return CachedImage{Image: ""}, err
	}
	return c.cache[imgPath], nil
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
		c.cache[imgPath] = cachedImage
		log.Debugf("added image '%s' to cache", imgPath)
	} else {
		return err
	}
	return nil
}
