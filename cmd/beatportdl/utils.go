package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func (app *application) background(fn func()) {
	app.wg.Add(1)

	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) backgroundCustom(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(fmt.Errorf("%s", err).Error())
			}
		}()
		fn()
	}()
}

func (app *application) downloadFile(url string, destination string) error {
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

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
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

func LogError(caller string, err error) {
	message := fmt.Sprintf("%s: %s", caller, err.Error())
	fmt.Println(message)
}

func FatalError(caller string, err error) {
	LogError(caller, err)
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
