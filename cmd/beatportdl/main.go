package main

import (
	"flag"
	"strings"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

const (
	configFilename = "beatportdl-config.yml"
	cacheFilename  = "beatportdl-credentials.json"
)

type application struct {
	config *config.AppConfig
	bp     *beatport.Beatport
	wg     sync.WaitGroup
	urls   []string
}

func main() {
	cfg, cachePath := Setup()

	bp := beatport.New(
		cfg.Username,
		cfg.Password,
		cachePath,
		cfg.Proxy,
	)

	if err := bp.LoadCachedTokenPair(); err != nil {
		if err := bp.NewTokenPair(); err != nil {
			FatalError("beatport", err)
		}
	}

	app := &application{
		config: cfg,
		bp:     bp,
	}

	flag.Parse()
	inputArgs := flag.Args()

	for _, arg := range inputArgs {
		if strings.HasSuffix(arg, ".txt") {
			app.parseTextFile(arg)
		} else {
			app.urls = append(app.urls, arg)
		}
	}

	for {
		if len(app.urls) == 0 {
			app.mainPrompt()
		}

		for _, url := range app.urls {
			app.background(func() {
				app.handleUrl(url)
			})
		}

		app.wg.Wait()
		app.urls = []string{}
	}
}
