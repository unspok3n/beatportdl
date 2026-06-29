package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

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
		switch {
		case errors.Is(err, io.EOF):
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "read input string: %v\n", err)
			os.Exit(1)
		}
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

func FindConfigFile() (string, bool, error) {
	var additionalDirs []string

	if runtime.GOOS == "linux" {
		var additionalDir string
		if xdgCfgHome, exists := os.LookupEnv("XDG_CONFIG_HOME"); exists {
			additionalDir = path.Join(xdgCfgHome, "beatportdl")
		} else {
			additionalDir = path.Join(os.Getenv("HOME"), ".config", "beatportdl")
		}
		additionalDirs = append(additionalDirs, additionalDir)
	}

	return findFile(configFilename, additionalDirs)
}

func FindCacheFile() (string, bool, error) {
	var additionalDirs []string

	if runtime.GOOS == "linux" {
		var additionalDir string
		if xdgCfgHome, exists := os.LookupEnv("XDG_STATE_HOME"); exists {
			additionalDir = path.Join(xdgCfgHome, "beatportdl")
		} else {
			additionalDir = path.Join(os.Getenv("HOME"), ".local", "state", "beatportdl")
		}
		additionalDirs = append(additionalDirs, additionalDir)
	}

	return findFile(cacheFilename, additionalDirs)
}

func FindErrorLogFile() (string, bool, error) {
	var additionalDirs []string
	return findFile(errorFilename, additionalDirs)
}

func findFile(fileName string, additionalDirs []string) (string, bool, error) {
	configFilePaths := []string{}

	filePathWorking, err := WorkingDirFilePath(fileName)
	if err != nil {
		return "", false, err
	}
	configFilePaths = append(configFilePaths, filePathWorking)

	filePathExec, err := ExecutableDirFilePath(fileName)
	if err != nil {
		return "", false, err
	}
	configFilePaths = append(configFilePaths, filePathExec)

	for _, addadditionalDir := range additionalDirs {
		additionalFilePath := path.Join(addadditionalDir, fileName)
		configFilePaths = append(configFilePaths, additionalFilePath)
	}

	for _, configFilePath := range configFilePaths {
		_, err = os.Stat(configFilePath)
		if err == nil {
			return configFilePath, true, nil
		}
	}

	// last entry in list is default
	defaultFilePath := configFilePaths[len(configFilePaths)-1]
	return defaultFilePath, false, nil
}

func CreateDirectory(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err := os.MkdirAll(directory, 0760); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}
	return nil
}
