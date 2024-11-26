package beatport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Beatport struct {
	username      string
	password      string
	tokenPair     *tokenPair
	cacheFilePath string
	client        *http.Client
	headers       map[string]string
	mutex         sync.RWMutex
}

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	IssuedAt     int64  `json:"issued_at"`
}

type Error struct {
	Detail *string `json:"detail,omitempty"`
	Error  *string `json:"error,omitempty"`
}

type Paginated[T any] struct {
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Count    int     `json:"count"`
	Page     string  `json:"page"`
	PerPage  int     `json:"per_page"`
	Results  []T     `json:"results"`
}

type Image struct {
	ID         int64  `json:"id"`
	URI        string `json:"uri"`
	DynamicURI string `json:"dynamic_uri"`
}

type ArtistType string

var (
	ArtistTypeMain     ArtistType = "main"
	ArtistTypeRemixers ArtistType = "remixers"
)

const (
	clientId = "ryZ8LuyQVPqbK2mBX2Hwt4qSMtnWuTYSqBPO92yQ"
)

const (
	baseUrl       = "https://api.beatport.com/v4"
	tokenEndpoint = "/auth/o/token/"
	authEndpoint  = "/auth/o/authorize/?client_id=" + clientId + "&response_type=code"
	loginEndpoint = "/auth/login/"
)

var (
	ErrInvalidAuthorizationCode = errors.New("invalid authorization code")
	ErrInvalidSessionCookie     = errors.New("invalid session cookie")
)

func New(username string, password string, cacheFilePath string, proxyUrl string) *Beatport {
	transport := &http.Transport{}
	if proxyUrl != "" {
		proxyURL, _ := url.Parse(proxyUrl)
		proxy := http.ProxyURL(proxyURL)
		transport.Proxy = proxy
	}
	headers := map[string]string{
		"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"accept-language": "en-US,en;q=0.9",
		"cache-control":   "max-age=0",
		"user-agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	}
	bp := Beatport{
		username:      username,
		password:      password,
		cacheFilePath: cacheFilePath,
		client: &http.Client{
			Timeout:   time.Duration(40) * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		headers: headers,
	}
	return &bp
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

	res, err := b.fetch("POST", tokenEndpoint, payload, "application/x-www-form-urlencoded")
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

func (b *Beatport) issueToken(code string) error {
	payload := map[string]string{
		"client_id": clientId,
	}

	if code != "" {
		payload["grant_type"] = "authorization_code"
		payload["code"] = code
	} else {
		payload["grant_type"] = "password"
		payload["username"] = b.username
		payload["password"] = b.password
	}

	res, err := b.fetch("POST", tokenEndpoint, payload, "application/x-www-form-urlencoded")
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

func (b *Beatport) authorize(sessionId string) (string, error) {
	b.headers["cookie"] = fmt.Sprintf("sessionid=%s", sessionId)
	res, err := b.fetch("GET", authEndpoint, nil, "")
	if err != nil {
		return "", err
	}
	delete(b.headers, "cookie")
	redirectUrl := res.Header.Get("Location")
	parsedUrl, err := url.Parse(redirectUrl)
	if err != nil {
		return "", err
	}
	query, _ := url.ParseQuery(parsedUrl.RawQuery)
	code := query.Get("code")
	if code != "" {
		return code, nil
	}
	return "", ErrInvalidAuthorizationCode
}

func (b *Beatport) login() (string, error) {
	payload := map[string]string{
		"username": b.username,
		"password": b.password,
	}

	res, err := b.fetch("POST", loginEndpoint, payload, "application/json")
	if err != nil {
		return "", err
	}
	cookies := res.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "sessionid" {
			return cookie.Value, nil
		}
	}
	return "", ErrInvalidSessionCookie
}

func (b *Beatport) NewTokenPair() error {
	fmt.Println("Logging in")
	sessionId, err := b.login()
	if err != nil {
		return fmt.Errorf("login: %v", err)
	}
	authorizationCode, err := b.authorize(sessionId)
	if err != nil {
		return fmt.Errorf("authorize: %v", err)
	}
	if err := b.issueToken(authorizationCode); err != nil {
		return fmt.Errorf("issue token: %v", err)
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

	if endpoint != tokenEndpoint && endpoint != authEndpoint && endpoint != loginEndpoint {
		currentTime := time.Now().Unix()

		b.mutex.RLock()
		tokenExpirationTime := b.tokenPair.IssuedAt + b.tokenPair.ExpiresIn
		b.mutex.RUnlock()
		if currentTime+300 >= tokenExpirationTime {
			b.mutex.Lock()
			fmt.Println("Refreshing token")
			_, err := b.refreshToken()
			if err != nil {
				if err := b.NewTokenPair(); err != nil {
					b.mutex.Unlock()
					return nil, fmt.Errorf("invalid token and authorization error: %w", err)
				}
			}
			b.mutex.Unlock()
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

	for key, value := range b.headers {
		req.Header.Add(key, value)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		if resp.StatusCode == http.StatusUnauthorized && endpoint != tokenEndpoint && endpoint != authEndpoint && endpoint != loginEndpoint {
			b.mutex.Lock()
			b.tokenPair.IssuedAt = 0
			b.mutex.Unlock()
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

func (i *Image) FormattedUrl(size string) string {
	return strings.Replace(
		i.DynamicURI,
		"{w}x{h}",
		size,
		-1,
	)
}
