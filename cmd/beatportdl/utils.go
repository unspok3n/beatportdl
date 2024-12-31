package main

import (
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func (app *application) background(fn func()) {
	app.wg.Add(1)

	go func() {
		app.semAcquire(app.globalSem)
		defer app.wg.Done()
		defer app.semRelease(app.globalSem)
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) downloadWorker(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)

	go func() {
		app.semAcquire(app.downloadSem)
		defer wg.Done()
		defer app.semRelease(app.downloadSem)
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) semAcquire(s chan struct{}) {
	s <- struct{}{}
}

func (app *application) semRelease(s chan struct{}) {
	<-s
}

func (app *application) downloadFile(url string, destination string, pbPrefix string) error {
	out, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if pbPrefix != "" {
		contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
		bar := app.pbp.AddBar(int64(contentLength), ProgressBarOptions(pbPrefix)...)

		proxyReader := bar.ProxyReader(resp.Body)
		defer proxyReader.Close()

		_, err = io.Copy(out, proxyReader)
		if err != nil {
			return err
		}
	} else {
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

func toMetaFunc(c *color.Color) func(string) string {
	return func(s string) string {
		return c.Sprint(s)
	}
}

func ProgressBarOptions(prefix string) []mpb.BarOption {
	red, green, blue := color.New(color.FgRed), color.New(color.FgGreen), color.New(color.FgBlue)

	options := []mpb.BarOption{
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.OnCompleteMeta(
				decor.OnComplete(
					decor.Meta(decor.Spinner([]string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}), toMetaFunc(red)),
					"✓ ",
				),
				toMetaFunc(green),
			),
			decor.Name(prefix, decor.WCSyncSpaceR),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Meta(decor.Percentage(decor.WCSyncSpace), toMetaFunc(blue)), ""),
			decor.OnComplete(decor.Name(" |"), ""),
			decor.OnComplete(decor.Meta(
				decor.EwmaSpeed(decor.SizeB1000(0), "% .2f", 30, decor.WCSyncSpace), toMetaFunc(blue),
			), ""),
			decor.OnComplete(decor.Name(" | "), ""),
			decor.OnComplete(decor.Meta(decor.EwmaETA(decor.ET_STYLE_MMSS, 0, decor.WCSyncWidth), toMetaFunc(blue)), ""),
		),
	}
	return options
}

func GetLine() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input string: %v\n", err)
		os.Exit(1)
	}
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, "\r")
	return input
}

func Pause() {
	fmt.Println("\nPress enter to exit")
	fmt.Scanln()
	os.Exit(1)
}

func (app *application) LogError(caller string, err error) {
	message := fmt.Sprintf("%s: %s\n", caller, err.Error())
	fmt.Fprint(app.logWriter, message)

	if app.logFile != nil {
		app.logFile.WriteString(message)
	}
}

func (app *application) LogInfo(info string) {
	message := fmt.Sprintf("%s\n", info)
	fmt.Fprint(app.logWriter, message)
}

func (app *application) FatalError(caller string, err error) {
	app.LogError(caller, err)
	Pause()
}

func ExecutableDirFilePath(fileName string) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	filePathExec := filepath.Join(execDir, fileName)
	return filePathExec, nil
}

func WorkingDirFilePath(fileName string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	filePathCurrent := filepath.Join(workingDir, fileName)
	return filePathCurrent, nil
}

func FindFile(fileName string) (string, error) {
	filePathExec, err := ExecutableDirFilePath(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	_, err = os.Stat(filePathExec)
	if err == nil {
		return filePathExec, nil
	}

	filePathCurrent, err := WorkingDirFilePath(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %v", err)
	}

	_, err = os.Stat(filePathCurrent)
	if err == nil {
		return filePathCurrent, nil
	}

	return "", fmt.Errorf("%s not found", fileName)
}

func CreateDirectory(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err := os.MkdirAll(directory, 0760); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}
	return nil
}
