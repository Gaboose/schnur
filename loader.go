package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

type Space int

const (
	SpaceLocal Space = iota + 1
	SpaceZoro
)

type Loader struct {
	Space   Space
	ZoroURL string
}

func (l *Loader) Load(ctx context.Context, url string) (*Resource, error) {
	rc, mime, err := l.readCloser(ctx, url)
	if err != nil {
		return nil, err
	}

	_, fileName := path.Split(url)

	ret := &Resource{
		ReadCloser: rc,
		FileName:   fileName,
		Mime:       mime,
	}

	if ret.IsJSON() {
		defer ret.Close()
		bts, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}

		ret.Bytes = bts
	}

	if *verbose {
		log.Println("Mime", mime)
		if ret.IsJSON() {
			log.Println("Body", string(ret.Bytes))
		}
	}

	return ret, nil
}

func (l *Loader) httpReadCloser(ctx context.Context, rawURL string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	return resp.Body, resp.Header.Get("content-type"), nil
}

func (l *Loader) fileReadCloser(path string) (io.ReadCloser, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}

	buf := bufio.NewReader(file)

	bts, err := buf.Peek(512)
	if err != nil && err != io.EOF {
		return nil, "", err
	}

	rc := struct {
		io.Reader
		io.Closer
	}{
		Reader: buf,
		Closer: file,
	}

	return rc, mimetype.Detect(bts).String(), nil
}

func (l *Loader) readCloser(ctx context.Context, rawURL string) (io.ReadCloser, string, error) {
	u, _ := url.Parse(rawURL)

	switch u.Scheme {
	case "http", "https":
		return l.httpReadCloser(ctx, rawURL)
	}

	switch l.Space {
	case SpaceLocal:
		return l.fileReadCloser(rawURL)
	case SpaceZoro:
		return l.httpReadCloser(ctx, l.ZoroURL+rawURL)
	default:
		panic("unexpected space")
	}
}

type Resource struct {
	io.ReadCloser
	Bytes    []byte
	FileName string
	Mime     string
}

func (r *Resource) IsVideo() bool {
	return strings.HasPrefix(r.Mime, "video")
}

func (r *Resource) IsJSON() bool {
	return strings.HasPrefix(r.Mime, "application/json")
}

func (r *Resource) save() (string, error) {
	if err := os.MkdirAll("./tmp", os.ModePerm); err != nil && err != os.ErrExist {
		return "", err
	}

	f, err := os.CreateTemp("./tmp", r.FileName)
	if err != nil {
		return "", err
	}

	go func() {
		if _, err := io.Copy(f, r); err != nil {
			panic(err)
		}
	}()

	return f.Name(), nil
}

func (r *Resource) PlayVideo(ctx context.Context) error {
	fname, err := r.save()
	if err != nil {
		return err
	}

	cmdParts := strings.Split(*videoCmd, " ")
	cmdParts = append(cmdParts, fname)

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	return cmd.Run()
}
