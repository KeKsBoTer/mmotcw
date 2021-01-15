package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// ImgCache is the global cache for all images
var ImgCache = PreviewCache{}

// Week stores information about the maimais, votes etc. of a week
type Week struct {
	Maimais       []Maimai
	CW            CW
	IsCurrentWeek bool
	Votes         []Vote
	CanVote       bool
	// template file name
	Template string
}

// SortMaimais sorts the maimais by date
func (w *Week) SortMaimais() {
	sort.Slice(w.Maimais[:], func(i, j int) bool {
		return w.Maimais[i].Time.After(w.Maimais[j].Time)
	})
}

func getMaiMaiPerCW(pathPrefix string, w string) (*Week, error) {
	cw, err := CWFromPath(w)
	if err != nil {
		return nil, err
	}
	imgFiles, err := GetImageFiles(w)
	if err != nil {
		return nil, err
	}
	week := Week{
		Maimais: []Maimai{},
		CW:      *cw,
	}
	for _, img := range imgFiles {
		if !strings.HasPrefix(img.Name(), "template.") {
			imgPath := cw.ImagePath(img.Name())
			cachedImage, err := ImgCache.GetImage(filepath.Join(w, img.Name()))
			if err != nil {
				log.Error(err)
				continue
			}
			week.Maimais = append(week.Maimais, Maimai{
				File:      img.Name(),
				Href:      filepath.Join(pathPrefix, imgPath),
				Time:      img.ModTime(),
				ImageSize: cachedImage.Size,
				Preview:   cachedImage.Preview,
			})
		}
	}
	week.SortMaimais()
	return &week, nil
}
