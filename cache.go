package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
)

// Base64String is a string that bolds base64 encoded content
type Base64String string

// CachedImage is a low dimensional chached version of an image
type CachedImage struct {
	Size    image.Point
	Preview Base64String
}

// PreviewCache is a key-value map with cached preview images
type PreviewCache map[string]CachedImage

// GetImage returns Base54 encoded image of path
// returns empty image if something goes wrong
func (c PreviewCache) GetImage(imgPath string) (CachedImage, error) {
	if cache, ok := c[imgPath]; ok {
		return cache, nil
	}
	err := c.cacheImage(imgPath)
	if err != nil {
		return CachedImage{Preview: ""}, err
	}
	return c[imgPath], nil
}

func (c PreviewCache) cacheImage(imgPath string) error {
	if imgFile, err := os.OpenFile(imgPath, os.O_RDONLY, os.ModePerm); err == nil {
		img, _, err := image.Decode(imgFile)
		if err != nil {
			return err
		}
		ratio := float32(img.Bounds().Size().Y) / float32(img.Bounds().Size().X)

		cachedImage := CachedImage{
			Size:    image.Point{330, 330},
			Preview: "",
		}

		cachedImage.Size.Y = int(ratio * float32(cachedImage.Size.X))

		// create smal image preview
		smallImage := resize.Resize(20, uint(ratio*20), img, resize.Lanczos3)
		buffer := bytes.NewBuffer([]byte{})
		if err := jpeg.Encode(buffer, smallImage, nil); err != nil {
			return err
		}
		cachedImage.Preview = Base64String(base64.RawStdEncoding.EncodeToString(buffer.Bytes()))
		c[imgPath] = cachedImage
	} else {
		return err
	}
	return nil
}
