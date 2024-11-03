package beatport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Beatport struct {
	username      string
	password      string
	tokenPair     *tokenPair
	cacheFilePath string
	client        *http.Client
}

type Error struct {
	Detail *string `json:"detail,omitempty"`
	Error  *string `json:"error,omitempty"`
}

type Image struct {
	ID         int64  `json:"id"`
	URI        string `json:"uri"`
	DynamicURI string `json:"dynamic_uri"`
}

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	IssuedAt     int64  `json:"issued_at"`
}

const (
	clientId     = "nBQh4XCUqE0cpoy609mC8GoyjCcJHBwbI374FYmE"
	clientSecret = "7oBWZwYOia9u4yblRmVTTet5sficrN7xbbCglbmRxoN08ShlpxyXbixLeov2wC62R3WsD2dxSTwLosi71FqpfLSOKnFSZ4FTXoayHNLHpWz7XcmyOMiLkqnbTPk2kI9L"
)

const (
	baseUrl      = "https://api.beatport.com/v4"
	authEndpoint = "/auth/o/token/"
)

func New(username string, password string, cacheFilePath string, proxyUrl string) (*Beatport, error) {
	transport := &http.Transport{}
	if proxyUrl != "" {
		proxyURL, _ := url.Parse(proxyUrl)
		proxy := http.ProxyURL(proxyURL)
		transport.Proxy = proxy
	}
	bp := Beatport{
		username:      username,
		password:      password,
		cacheFilePath: cacheFilePath,
		client: &http.Client{
			Timeout:   time.Duration(40) * time.Second,
			Transport: transport,
		},
	}

	return &bp, nil
}

func (b *Beatport) LoadCachedTokenPair() error {
	data, err := os.ReadFile(b.cacheFilePath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var loadedToken tokenPair
	if err := json.Unmarshal(data, &loadedToken); err != nil {
		return fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	b.tokenPair = &loadedToken

	return nil
}

func (b *Beatport) cacheTokenPair() error {
	data, err := json.MarshalIndent(b.tokenPair, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokenPair: %w", err)
	}

	if err := os.WriteFile(b.cacheFilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token to cache: %w", err)
	}

	return nil
}

func (b *Beatport) refreshToken() (*tokenPair, error) {
	payload := map[string]string{
		"client_id":     clientId,
		"refresh_token": b.tokenPair.RefreshToken,
		"grant_type":    "refresh_token",
	}

	res, err := b.fetch("POST", authEndpoint, payload, "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &tokenPair{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	b.tokenPair = response
	b.tokenPair.IssuedAt = time.Now().Unix()
	b.cacheTokenPair()

	return response, nil
}

func (b *Beatport) Authorize() error {
	payload := map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"grant_type":    "password",
		"username":      b.username,
		"password":      b.password,
	}

	res, err := b.fetch("POST", authEndpoint, payload, "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	response := &tokenPair{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return err
	}
	b.tokenPair = response
	b.tokenPair.IssuedAt = time.Now().Unix()
	err = b.cacheTokenPair()
	if err != nil {
		return err
	}

	return nil
}

func encodeFormPayload(payload interface{}) (url.Values, error) {
	values := url.Values{}

	switch p := payload.(type) {
	case map[string]string:
		for key, value := range p {
			values.Set(key, value)
		}
	case url.Values:
		values = p
	default:
		return nil, errors.New("invalid payload")
	}

	return values, nil
}

func (b *Beatport) fetch(method, endpoint string, payload interface{}, contentType string) (*http.Response, error) {
	var body bytes.Buffer

	if endpoint != authEndpoint {
		currentTime := time.Now().Unix()
		tokenExpirationTime := b.tokenPair.IssuedAt + b.tokenPair.ExpiresIn
		if currentTime+300 >= tokenExpirationTime {
			fmt.Println("Refreshing token")
			_, err := b.refreshToken()
			if err != nil {
				fmt.Println("Authorizing")
				if err := b.Authorize(); err != nil {
					return nil, fmt.Errorf("invalid token and authorization error: %w", err)
				}
			}
		}
	}

	if payload != nil {
		switch contentType {
		case "application/json":
			if err := json.NewEncoder(&body).Encode(payload); err != nil {
				return nil, fmt.Errorf("failed to encode json payload: %w", err)
			}
		case "application/x-www-form-urlencoded":
			formData, err := encodeFormPayload(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to encode form payload: %w", err)
			}
			body.WriteString(formData.Encode())
		default:
			return nil, fmt.Errorf("unsupported content type: %s", contentType)
		}
	}

	req, err := http.NewRequest(method, baseUrl+endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if b.tokenPair != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.tokenPair.AccessToken))
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			b.tokenPair.IssuedAt = 0
			return b.fetch(method, endpoint, payload, contentType)
		}
		defer resp.Body.Close()
		response := &Error{}
		if err = json.NewDecoder(resp.Body).Decode(response); err == nil {
			detail := "Unknown error"
			if response.Detail != nil {
				detail = *response.Detail
			} else if response.Error != nil {
				detail = *response.Error
			}
			return nil, fmt.Errorf(
				"request failed with status code: %d - %s",
				resp.StatusCode,
				detail,
			)
		}
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp, nil
}
