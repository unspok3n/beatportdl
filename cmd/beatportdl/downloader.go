package main

import (
	"fmt"
	"os"
)

func (app *application) prepareTrack(id int64) error {
	track, err := app.bp.GetTrack(id)
	if err != nil {
		return err
	}
	fmt.Printf("Downloading %s (%s)\n", track.Name, track.MixName)
	stream, err := app.bp.DownloadTrack(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(app.config.DownloadsDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(app.config.DownloadsDirectory, 0760); err != nil {
			return err
		}
	}
	fileName := track.Filename()
	filePath := fmt.Sprintf("%s/%s", app.config.DownloadsDirectory, fileName)
	if err = app.downloadFile(stream.Location, filePath); err != nil {
		return err
	}
	fmt.Printf("Finished downloading %s (%s)\n", track.Name, track.MixName)

	return nil
}
