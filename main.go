package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
)

type Spec struct {
	Search *Search `json:"search"`
	List   *List   `json:"list"`
	URL    *string `json:"url"`
}

func execURL(ctx context.Context, url string, l *Loader) (string, error) {
	if *verbose {
		log.Println("URL", url)
	}

	rc, err := l.Load(ctx, url)
	if err != nil {
		return "", fmt.Errorf("load: %w", err)
	}
	defer rc.Close()

	if rc.IsVideo() {
		err := rc.PlayVideo(ctx)
		return "", err
	}

	var spec Spec

	if err := json.Unmarshal(rc.Bytes, &spec); err != nil {
		return "", fmt.Errorf("body: unmarshal: %w", err)
	}

	var nextURL string

	switch {
	case spec.Search != nil:
		nextURL, err = spec.Search.Run(ctx, l)
		if err != nil {
			return "", fmt.Errorf("search: %w", err)
		}
	case spec.List != nil:
		nextURL, err = spec.List.Run(ctx)
		if err != nil {
			return "", fmt.Errorf("list: %w", err)
		}
	case spec.URL != nil:
		return *spec.URL, nil
	}

	return nextURL, nil
}

var (
	zoroURLFlag = flag.String("zoro-url", "http://zoro.fly.dev/", "")
	spaceFlag   = flag.String("space", "zoro", `"zoro" or "local"`)
	verbose     = flag.Bool("v", false, "print more details to stderr")
	videoCmd    = flag.String("video-cmd", "vlc", "command to play video with")

	zoroURL *url.URL
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	var err error
	zoroURL, err = url.Parse(*zoroURLFlag)
	if err != nil {
		panic(err)
	}

	loader := &Loader{
		ZoroURL: *zoroURLFlag,
	}

	switch *spaceFlag {
	case "local":
		loader.Space = SpaceLocal
	case "zoro":
		loader.Space = SpaceZoro
	}

	ctx := context.Background()
	nextURL := flag.Arg(0)

	for {
		nextURL, err = execURL(ctx, nextURL, loader)
		if err != nil {
			panic(err)
		}

		if nextURL == "" {
			break
		}
	}
}
