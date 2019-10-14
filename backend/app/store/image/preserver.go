package image

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
)

// Preserver provides a CommentConverter that store all external images and uses
// internal urls instead.
type Preserver struct {
	Enabled      bool
	RemarkURL    string
	ImageService *Service
	Timeout      time.Duration
}

// Convert extracts all external images, stores them internally and replaces
// urls with stored versions. If for some reason it can't store some image original
// url will be used.
func (p Preserver) Convert(commentHTML string, userID string) string {
	if !p.Enabled {
		return commentHTML
	}

	imgs, err := p.extractExternalImages(commentHTML)
	if err != nil {
		return commentHTML
	}

	return p.downloadAndReplaceExternalImages(commentHTML, userID, imgs)
}

func (p Preserver) extractExternalImages(commentHTML string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return nil, errors.Wrap(err, "can't create document")
	}
	result := []string{}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if im, ok := s.Attr("src"); ok {
			if !strings.HasPrefix(im, p.RemarkURL) {
				result = append(result, im)
			}
		}
	})
	return result, nil
}

// tries to download and save image, returns saved image ID if successful
func (p Preserver) downloadAndSaveImage(userID string, imgURL string) (string, error) {
	log.Printf("[DEBUG] downloading image %s", imgURL)

	timeout := 60 * time.Second // default
	if p.Timeout > 0 {
		timeout = p.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := http.Client{Timeout: 30 * time.Second}
	var resp *http.Response
	err := repeater.NewDefault(5, time.Second).Do(ctx, func() error {
		var e error
		req, e := http.NewRequest("GET", imgURL, nil)
		if e != nil {
			return errors.Wrapf(e, "failed to make request for %s", imgURL)
		}
		resp, e = client.Do(req.WithContext(ctx))
		return e
	})
	if err != nil {
		log.Print(err.Error())
		return "", err
	}
	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] can't close body, %s", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("got unsuccessful response status %d while fetching %s", resp.StatusCode, imgURL)
	}

	imgID, err := p.ImageService.Save("external.unknown", userID, resp.Body)
	if err != nil {
		return "", err
	}
	return imgID, nil
}

func (p Preserver) downloadAndReplaceExternalImages(commentHTML string, userID string, externalImages []string) string {
	for _, img := range externalImages {
		imgID, err := p.downloadAndSaveImage(userID, img)
		resImgURL := p.RemarkURL + "/api/v1/picture/" + imgID
		if err != nil {
			log.Printf("[WARN] unable to preserve image: %s. Using original url: %s", err.Error(), img)
			resImgURL = img
		}
		commentHTML = strings.Replace(commentHTML, img, resImgURL, -1)
	}
	return commentHTML
}
