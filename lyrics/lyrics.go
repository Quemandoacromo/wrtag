package lyrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/cascadia"
	"go.senan.xyz/wrtag/clientutil"
	"golang.org/x/net/html"
)

var ErrLyricsNotFound = errors.New("lyrics not found")

type Source interface {
	Search(ctx context.Context, artist, song string) (string, error)
}

type FastestSource []Source

func (fsrc FastestSource) Search(ctx context.Context, artist, song string) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lyricData := make(chan string)
	for _, src := range fsrc {
		go func() {
			ld, _ := src.Search(ctx, artist, song)
			lyricData <- ld
		}()
	}

	for ld := range lyricData {
		if ld != "" {
			return ld, nil
		}
	}
	return "", ErrLyricsNotFound
}

var musixmatchBaseURL = `https://www.musixmatch.com/lyrics`
var musixmatchSelectContent = cascadia.MustCompile(`div.r-1v1z2uz:nth-child(1)`)
var musixmatchIgnore = []string{"Still no lyrics here"}
var musixmatchEsc = strings.NewReplacer(
	" ", "-",
	"(", "",
	")", "",
	"[", "",
	"]", "",
)

type Musixmatch struct {
	RateLimit time.Duration

	initOnce   sync.Once
	HTTPClient *http.Client
}

func (mm *Musixmatch) Search(ctx context.Context, artist, song string) (string, error) {
	mm.initOnce.Do(func() {
		mm.HTTPClient = clientutil.WrapClient(mm.HTTPClient, clientutil.Chain(
			clientutil.WithRateLimit(mm.RateLimit),
		))
	})

	url, _ := url.Parse(musixmatchBaseURL)
	url = url.JoinPath(musixmatchEsc.Replace(artist))
	url = url.JoinPath(musixmatchEsc.Replace(song))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	resp, err := mm.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("req page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return "", ErrLyricsNotFound
	}

	node, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse page: %w", err)
	}

	var out strings.Builder
	iterText(cascadia.Query(node, musixmatchSelectContent), func(s string) {
		out.WriteString(s + "\n")
	})
	for _, ig := range musixmatchIgnore {
		if strings.Contains(out.String(), ig) {
			return "", nil
		}
	}
	return out.String(), nil
}

var geniusBaseURL = `https://genius.com`
var geniusSelectContent = cascadia.MustCompile(`div[class^="Lyrics__Container-"]`)
var geniusEsc = strings.NewReplacer(
	" ", "-",
	"(", "",
	")", "",
	"[", "",
	"]", "",
)

type Genius struct {
	RateLimit time.Duration

	initOnce   sync.Once
	HTTPClient *http.Client
}

func (g *Genius) Search(ctx context.Context, artist, song string) (string, error) {
	g.initOnce.Do(func() {
		g.HTTPClient = clientutil.WrapClient(g.HTTPClient, clientutil.Chain(
			clientutil.WithRateLimit(g.RateLimit),
		))
	})

	// use genius case rules to miminise redirects
	page := fmt.Sprintf("%s-%s-lyrics", artist, song)
	page = strings.ToUpper(string(page[0])) + strings.ToLower(page[1:])

	url, _ := url.Parse(geniusBaseURL)
	url = url.JoinPath(geniusEsc.Replace(page))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("req page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return "", ErrLyricsNotFound
	}

	node, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse page: %w", err)
	}

	var out strings.Builder
	iterText(cascadia.Query(node, geniusSelectContent), func(s string) {
		out.WriteString(s + "\n")
	})
	return out.String(), nil
}

func iterText(n *html.Node, f func(string)) {
	if n == nil {
		return
	}
	if n.Type == html.TextNode {
		f(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		iterText(c, f)
	}
}
