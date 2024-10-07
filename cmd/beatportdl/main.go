package main

import (
	"flag"
	"fmt"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

const authUrl = "https://api.beatport.com/v4/auth/o/authorize/?client_id=ryZ8LuyQVPqbK2mBX2Hwt4qSMtnWuTYSqBPO92yQ&response_type=code"

type application struct {
	config *config.AppConfig
	bp     *beatport.Beatport
	wg     sync.WaitGroup
}

func main() {
	config, err := config.ParseConfig("beatportdl-config.yml")
	if err != nil {
		FatalError("load config", err)
	}

	authorizeFlag := flag.Bool("authorize", false, "Start the authorization process")

	flag.Parse()

	bpClient, err := beatport.New(config.Proxy, !*authorizeFlag)
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

		Pause()
	}

	app := &application{
		config: config,
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
