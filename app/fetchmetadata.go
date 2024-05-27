package main

import (
	"fmt"
	"net/http"
	"net/url"

	"io"

	"github.com/asaskevich/govalidator"
	"golang.org/x/net/html"
)

type HTMLMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
}

func fetchMetadata(msg string) (preview *HTMLMeta, isLink bool, err error) {

	// rw.Header().Set(`Content-Type`, `application/json`)
	// err := req.ParseForm()
	// if err != nil {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	json.NewEncoder(rw).Encode(map[string]string{"error": err.Error()})
	// 	return
	// }

	// link := req.FormValue(`link`)
	// if link == "" {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	json.NewEncoder(rw).Encode(map[string]string{"error": `empty value of link`})
	// 	return
	// }
	if !validateURL(msg) {
		return preview, false, nil
	}

	if _, err = url.Parse(msg); err != nil {
		fmt.Printf("error Parse %v", msg)
		return preview, false, err
	}

	resp, err := http.Get(msg)
	if err != nil {
		fmt.Printf("error Get %v", err)
		return preview, false, err
	}
	defer resp.Body.Close()

	preview = extract(resp.Body)
	return preview, true, nil
}

func extract(resp io.Reader) *HTMLMeta {
	z := html.NewTokenizer(resp)
	titleFound := false

	hm := new(HTMLMeta)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return hm
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			if t.Data == `body` {
				return hm
			}
			if t.Data == "title" {
				titleFound = true
			}
			if t.Data == "meta" {
				desc, ok := extractMetaProperty(t, "description")
				if ok {
					hm.Description = desc
				}

				ogTitle, ok := extractMetaProperty(t, "og:title")
				if ok {
					hm.Title = ogTitle
				}

				ogDesc, ok := extractMetaProperty(t, "og:description")
				if ok {
					hm.Description = ogDesc
				}

				ogImage, ok := extractMetaProperty(t, "og:image")
				if ok {
					hm.Image = ogImage
				}

				ogSiteName, ok := extractMetaProperty(t, "og:site_name")
				if ok {
					hm.SiteName = ogSiteName
				}
			}
		case html.TextToken:
			if titleFound {
				t := z.Token()
				hm.Title = t.Data
				titleFound = false
			}
		}
	}
	return hm
}

func extractMetaProperty(t html.Token, prop string) (content string, ok bool) {
	for _, attr := range t.Attr {
		if attr.Key == "property" && attr.Val == prop {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}

	return
}

func validateURL(u string) bool {
	return govalidator.IsRequestURL(u)
}
