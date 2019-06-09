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
	"sync"
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
	var netClient = &http.Client{}
	req, _ := http.NewRequest("GET", URL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")

	resp, err := netClient.Do(req)
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

func getArticle(s *goquery.Selection, item *feeds.Item, sucess *int, name string, wg *sync.WaitGroup) {
	title := strings.TrimSpace(s.Find(".item_title").Text())
	URL, exist := s.Find(".item_title > a").Attr("href")
	if !exist {
		msg := fmt.Sprintf("Cannot find URL for title %s", title)
		log.Print(msg)
		return
	}
	log.Printf("Title: %s, URL: %s", title, URL)

	doc, err := getDocument(URL)
	if err != nil {
		log.Printf("Failed to get document for URL: %s\n%s", URL, err)
		return
	}
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}`)
	timeStr := re.FindString(doc.Find("#Main > .box > .header > small").Text())
	time, _ := time.Parse("2006-01-02 15:04", timeStr)
	// Process img tag.
	doc.Find("#js_content img").Each(func(i int, s *goquery.Selection) {
		src := s.AttrOr("data-src", "")
		if src == "" {
			return
		}
		startPos := strings.LastIndex(src, "http")
		if startPos >= 0 {
			src = src[startPos:]
		}
		s.SetAttr("src", src)
		// Remove all data-* attribute
		toRemoveAttr := []string{
			"data-label",
			"data-backh",
			"data-backw",
			"data-before-oversubscription-url",
			"data-ratio",
			"data-src",
			"data-type",
			"data-w",
			"data-copyright",
			"data-s",
			"data-ad-layout",
			"data-ad-format",
			"data-ad-client",
			"data-ad-slot",
			"data-croporisrc",
			"data-cropx1",
			"data-cropx2",
			"data-cropy1",
			"data-cropy2",
			"data-role",
			"data-id",
			"data-width",
			"data-cropselx1",
			"data-cropselx2",
			"data-cropsely1",
			"data-cropsely2",
			"data-style-type",
			"data-url",
			"data-author-name",
			"data-content-utf8-length",
			"data-source-title",
			"data-original-title",
			"data-autoskip",
			"data-oversubscription-url",
		}
		for _, attr := range toRemoveAttr {
			s.RemoveAttr(attr)
		}
	})

	contentHTML, _ := doc.Find("#js_content").Html()
	*item = feeds.Item{
		Title:       title,
		Link:        &feeds.Link{Href: URL},
		Author:      &feeds.Author{Name: name, Email: ""},
		Id:          URL,
		Created:     time,
		Description: contentHTML,
	}
	defer wg.Done()
}

func handleJtks(w http.ResponseWriter, r *http.Request, config *config) {
	log.Printf("Fetching name: %s URL: %s", config.Name, config.URL)
	doc, err := getDocument(config.URL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	now := time.Now()
	fullURL := fmt.Sprintf("%s%s", r.Host, r.RequestURI)
	feed := &feeds.Feed{
		Title:   config.Name,
		Link:    &feeds.Link{Href: fullURL},
		Id:      fullURL,
		Created: now,
	}
	feed.Items = make([]*feeds.Item, 10)
	for i := 0; i < 10; i++ {
		feed.Items[i] = &feeds.Item{}
	}
	success := make([]int, 10)
	var wg sync.WaitGroup

	doc.Find("#Main > .box > .entries > .item table tbody tr").Each(func(i int, s *goquery.Selection) {
		if i >= 10 {
			return
		}

		sleepTime, _ := time.ParseDuration("1s")
		time.Sleep(sleepTime)
		wg.Add(1)
		go getArticle(s, feed.Items[i], &success[i], config.Name, &wg)
	})

	wg.Wait()

	rss, err := feed.ToAtom()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(rss))
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(r.URL.Path, "/")
	log.Println(p)
	if len(p) < 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := p[2]
	if strings.HasSuffix(name, ".xml") {
		name = strings.TrimSuffix(name, ".xml")
	}
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

	switch config.Source {
	case "jtks":
		handleJtks(w, r, &config)
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Unknown source: %s", config.Source)))
	}

	log.Println("Done")
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
	listenURL := fmt.Sprintf("0.0.0.0:%s", port)

	// Register handlers
	http.HandleFunc("/rss/", handler)
	log.Printf("Listening on %s...", listenURL)
	log.Fatal(http.ListenAndServe(listenURL, nil))
}
