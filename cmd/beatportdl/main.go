package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

const authUrl = "https://api.beatport.com/v4/auth/o/authorize/?client_id=ryZ8LuyQVPqbK2mBX2Hwt4qSMtnWuTYSqBPO92yQ&response_type=code"
const (
	configFilename = "beatportdl-config.yml"
	cacheFilename  = "beatportdl-credentials.json"
)

type application struct {
	config *config.AppConfig
	bp     *beatport.Beatport
	wg     sync.WaitGroup
}

func main() {
	configFilePath, err := FindFile(configFilename)
	if err != nil {
		FatalError("find config file", err)
	}
	parsedConfig, err := config.ParseConfig(configFilePath)
	if err != nil {
		FatalError("load config", err)
	}

	authorizeFlag := flag.Bool("authorize", false, "Start the authorization process")

	flag.Parse()

	execCachePath, err := ExecutableDirFilePath(cacheFilename)
	if err != nil {
		FatalError("get executable path", err)
	}
	cacheFilePath := execCachePath

	if !*authorizeFlag {
		_, err = os.Stat(cacheFilePath)
		if err != nil {
			workingCachePath, err := WorkingDirFilePath(cacheFilename)
			if err != nil {
				FatalError("get current working dir", err)
			}
			_, err = os.Stat(workingCachePath)
			if err != nil {
				*authorizeFlag = true
			} else {
				cacheFilePath = workingCachePath
			}
		}
	}

	bpClient, err := beatport.New(cacheFilePath, parsedConfig.Proxy)
	if err != nil {
		FatalError("beatport api client", err)
	}

	if *authorizeFlag {
		message := `In your browser, open the following url, login if necessary, and then copy the "code" parameter from the address bar`
		fmt.Println(message)
		fmt.Print(authUrl + "\n\n")
		fmt.Print("Enter authorization code: ")
		code := GetLine()

		if _, err := bpClient.Authorize(code); err != nil {
			FatalError("beatport", err)
		}

		fmt.Println("Successfully authorized!")
	}

	if err := bpClient.LoadCachedTokenPair(); err != nil {
		FatalError("load cached token pair", err)
	}

	app := &application{
		config: parsedConfig,
		bp:     bpClient,
	}

	fmt.Print("Enter track or release link: ")
	input := GetLine()

	link, err := app.bp.ParseUrl(input)
	if err != nil {
		FatalError("parse url", err)
	}

	if link.Type == beatport.BeatportTrackLink {
		if err := app.prepareTrack(link.ID); err != nil {
			FatalError("download track", err)
		}
	} else if link.Type == beatport.BeatportReleaseLink {
		release, err := app.bp.GetRelease(link.ID)
		if err != nil {
			FatalError("fetch release", err)
		}

		for _, trackUrl := range release.TrackUrls {
			app.background(func() {
				trackLink, _ := app.bp.ParseUrl(trackUrl)
				if err := app.prepareTrack(trackLink.ID); err != nil {
					panic(err)
				}
			})
		}

		app.wg.Wait()
	}

	Pause()
}
