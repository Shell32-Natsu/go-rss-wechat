package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
)

type config struct {
	Name   string `json:"Name"`
	URL    string `json:"Url"`
	Source string `json:"Source"`
}

var configs []config

func getDocument(URL string) (*goquery.Document, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to get %s", URL)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s\nBody:\n%s", resp.StatusCode, resp.Status, string(body))
	}
	return goquery.NewDocumentFromReader(strings.NewReader(string(body)))
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(r.URL.Path, "/")
	log.Println(p)
	if len(p) < 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := p[2]
	var config config
	foundConfig := false
	for _, config = range configs {
		if config.Name == name {
			foundConfig = true
			break
		}
	}

	if !foundConfig {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Printf("Fetching name: %s URL: %s", config.Name, config.URL)
	doc, err := getDocument(config.URL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	now := time.Now()
	feed := &feeds.Feed{
		Title:   config.Name,
		Link:    &feeds.Link{Href: r.URL.String()},
		Created: now,
	}
	doc.Find("#Main > .box > .entries > .item table tbody tr").Each(func(i int, s *goquery.Selection) {
		if i > 10 {
			return
		}

		title := strings.TrimSpace(s.Find(".item_title").Text())
		URL, exist := s.Find(".item_title > a").Attr("href")
		if !exist {
			msg := fmt.Sprintf("Cannot find URL for title %s", title)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(msg))
			return
		}
		log.Printf("Title: %s, URL: %s", title, URL)

		doc, err := getDocument(URL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}`)
		timeStr := re.FindString(doc.Find("#Main > .box > .header > small").Text())
		time, _ := time.Parse("2006-01-02 15:04", timeStr)
		contentHTML, _ := doc.Find("#js_content").Html()
		feed.Items = append(feed.Items, &feeds.Item{
			Title:   title,
			Link:    &feeds.Link{Href: URL},
			Author:  &feeds.Author{Name: config.Name, Email: ""},
			Id:      URL,
			Created: time,
			Content: contentHTML,
		})
	})

	rss, err := feed.ToRss()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(rss))
}

func main() {
	// Read config file
	seeds, err := ioutil.ReadFile("./seeds.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(seeds, &configs)
	if err != nil {
		panic(err)
	}

	for idx, config := range configs {
		log.Printf("%d: %s %s %s", idx, config.Name, config.URL, config.Source)
	}
	port := os.Args[1]
	listenURL := fmt.Sprintf("127.0.0.1:%s", port)

	// Register handlers
	http.HandleFunc("/rss/", handler)
	log.Printf("Listening on %s...", listenURL)
	log.Fatal(http.ListenAndServe(listenURL, nil))
}
