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
	tokenPair *tokenPair
	client    *http.Client
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
	credentialsCacheFile = "beatportdl-credentials.json"
	clientId             = "ryZ8LuyQVPqbK2mBX2Hwt4qSMtnWuTYSqBPO92yQ"
)

const (
	baseUrl      = "https://api.beatport.com/v4"
	authEndpoint = "/auth/o/token/"
)

func New(proxyUrl string, readCache bool) (*Beatport, error) {
	transport := &http.Transport{}
	if proxyUrl != "" {
		proxyURL, _ := url.Parse(proxyUrl)
		proxy := http.ProxyURL(proxyURL)
		transport.Proxy = proxy
	}
	bp := Beatport{
		client: &http.Client{
			Timeout:   time.Duration(40) * time.Second,
			Transport: transport,
		},
	}

	if readCache {
		if err := bp.loadCachedTokenPair(); err != nil {
			return nil, fmt.Errorf("failed to load credentials from cache: %v", err)
		}
	}

	return &bp, nil
}

func (b *Beatport) loadCachedTokenPair() error {
	data, err := os.ReadFile(credentialsCacheFile)
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

	if err := os.WriteFile(credentialsCacheFile, data, 0600); err != nil {
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

func (b *Beatport) Authorize(code string) (*tokenPair, error) {
	payload := map[string]string{
		"client_id":  clientId,
		"code":       code,
		"grant_type": "authorization_code",
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
			b.refreshToken()
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
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp, nil
}
