package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
		log.Fatalf("Error reading input string: %v", err)
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

func FatalError(caller string, err error) {
	message := fmt.Sprintf("%s: %s", caller, err.Error())
	fmt.Println(message)
	Pause()
}
