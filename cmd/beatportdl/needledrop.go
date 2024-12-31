package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/grafov/m3u8"
	"github.com/vbauerster/mpb/v8"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type StreamKey struct {
	Value []byte
	IV    []byte
}

func getStreamSegments(stream string) (*[]string, *StreamKey, error) {
	resp, err := http.Get(stream)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}
	playlist, _, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, nil, err
	}
	u, err := url.Parse(stream)
	if err != nil {
		return nil, nil, err
	}
	base := u.Scheme + "://" + u.Host + path.Dir(u.Path) + "/"
	media := playlist.(*m3u8.MediaPlaylist)
	var segments []string
	var streamKey StreamKey
	for i, segment := range media.Segments {
		if segment == nil {
			break
		}
		if i == 0 {
			req, err := http.Get(base + segment.Key.URI)
			if err != nil {
				return nil, nil, err
			}
			defer req.Body.Close()
			if req.StatusCode != http.StatusOK {
				return nil, nil, fmt.Errorf("get stream key failed with status code: %d", req.StatusCode)
			}
			keyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, nil, fmt.Errorf("read stream key: %v", err)
			}
			ivBytes, err := hex.DecodeString(strings.TrimPrefix(segment.Key.IV, "0x"))
			if err != nil {
				return nil, nil, fmt.Errorf("decode stream iv: %v", err)
			}
			streamKey.Value = keyBytes
			streamKey.IV = ivBytes
		}
		segments = append(segments, base+segment.URI)
	}

	return &segments, &streamKey, nil
}

func decryptSegment(segment []byte, key StreamKey) ([]byte, error) {
	block, err := aes.NewCipher(key.Value)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(block, key.IV)
	decrypted := make([]byte, len(segment))
	cbc.CryptBlocks(decrypted, segment)
	padding := decrypted[len(decrypted)-1]
	return decrypted[:len(decrypted)-int(padding)], nil
}

func (app *application) downloadSegments(path string, segmentUrls []string, key StreamKey, pbPrefix string) (string, error) {
	tempFileName := uuid.New().String()
	path = filepath.Join(path, tempFileName)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return "", err
	}

	var bar *mpb.Bar
	var start time.Time
	if pbPrefix != "" {
		start = time.Now()
		total := len(segmentUrls)
		bar = app.pbp.AddBar(int64(total), ProgressBarOptions(pbPrefix)...)
	}

	for _, segmentUrl := range segmentUrls {
		req, err := http.Get(segmentUrl)
		if err != nil {
			return "", err
		}
		defer req.Body.Close()
		if req.StatusCode != http.StatusOK {
			return "", errors.New(req.Status)
		}
		segBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return "", err
		}
		decSegBytes, err := decryptSegment(segBytes, key)
		if err != nil {
			return "", err
		}
		_, err = file.Write(decSegBytes)
		if err != nil {
			return "", err
		}
		if bar != nil {
			bar.EwmaIncrInt64(1, time.Since(start))
		}
	}
	err = file.Close()
	if err != nil {
		return "", err
	}
	return path, nil
}

func remuxToM4A(input, output string) error {
	cmd := exec.Command("ffmpeg",
		"-i", input,
		"-map_metadata", "-1",
		"-c:a", "copy",
		output,
	)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}
