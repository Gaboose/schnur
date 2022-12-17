package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	schemeZoro  = "zoro"
	schemeZoros = "zoros"
)

type Spec struct {
	Search *Search `json:"search"`
	List   *List   `json:"list"`
}

func resolveURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}

	if u.Scheme == schemeZoro {
		u.Scheme = "http"
		return *zoroFlag + u.String(), nil
	} else if u.Scheme == schemeZoros {
		u.Scheme = "https"
		return *zoroFlag + u.String(), nil
	}

	return rawURL, nil
}

func execURL(ctx context.Context, url string) (string, error) {
	log.Printf("Resvoling url: %s\n", url)
	url, err := resolveURL(url)
	if err != nil {
		return "", fmt.Errorf("resolve url: %w", err)
	}

	log.Printf("Resolved url: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("body: read: %w", err)
	}

	log.Printf("Received from %s: %s\n", url, string(bts))

	var spec Spec

	if err := json.Unmarshal(bts, &spec); err != nil {
		return "", fmt.Errorf("body: unmarshal: %w", err)
	}

	var nextURL string

	switch {
	case spec.Search != nil:
		nextURL, err = spec.Search.Run(ctx)
		if err != nil {
			return "", fmt.Errorf("search: %w", err)
		}
	case spec.List != nil:
		nextURL, err = spec.List.Run(ctx)
		if err != nil {
			return "", fmt.Errorf("list: %w", err)
		}
	}

	return nextURL, nil
}

var (
	zoroFlag = flag.String("zoro", "http://localhost:8080/", "")
	zoroURL  *url.URL
)

func main() {
	urlFlag := flag.String("url", "", "")
	flag.Parse()

	if *urlFlag == "" || *zoroFlag == "" {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	zoroURL, err = url.Parse(*zoroFlag)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	nextURL := *urlFlag

	for {
		nextURL, err = execURL(ctx, nextURL)
		if err != nil {
			panic(err)
		}

		log.Println("DEBUG nextURL", nextURL)

		if nextURL == "" {
			break
		}
	}

	// grid := tview.NewGrid().
	// 	// SetRows(3, 0, 3).
	// 	// SetColumns(30, 0, 30).
	// 	SetBorders(true).
	// 	AddItem(inputField, 0, 0, 1, 1, 0, 0, false).
	// 	AddItem(list, 1, 0, 1, 1, 0, 0, true)

	// if err := tview.NewApplication().SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
	// 	panic(err)
	// }

	// if err := app.SetRoot(inputField, true).SetFocus(inputField).Run(); err != nil {
	// 	panic(err)
	// }
}
