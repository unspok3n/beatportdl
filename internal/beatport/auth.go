package beatport

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"os"
	"sync"
	"time"
)

const (
	clientId      = "ryZ8LuyQVPqbK2mBX2Hwt4qSMtnWuTYSqBPO92yQ"
	tokenEndpoint = "/auth/o/token/"
	authEndpoint  = "/auth/o/authorize/?client_id=" + clientId + "&response_type=code"
	loginEndpoint = "/auth/login/"
)

var (
	ErrInvalidAuthorizationCode = errors.New("invalid authorization code")
	ErrInvalidSessionCookie     = errors.New("invalid session cookie")
	ErrLoginIDMismatch          = errors.New("login id does not match")
)

type Auth struct {
	username  string
	password  string
	tokenPair *tokenPair
	cacheFile string
	mutex     sync.RWMutex
}

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	LoginID      string `json:"login_id"`
	IssuedAt     int64  `json:"issued_at"`
}

func NewAuth(username, password, cacheFile string) *Auth {
	return &Auth{
		username:  username,
		password:  password,
		cacheFile: cacheFile,
	}
}

func (a *Auth) LoadCache() error {
	data, err := os.ReadFile(a.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var loadedToken tokenPair
	if err := json.Unmarshal(data, &loadedToken); err != nil {
		return fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	if loadedToken.LoginID != a.loginId() {
		return ErrLoginIDMismatch
	}

	a.tokenPair = &loadedToken

	return nil
}

func (a *Auth) WriteCache() error {
	data, err := json.MarshalIndent(a.tokenPair, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokenPair: %w", err)
	}

	if err := os.WriteFile(a.cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token to cache: %w", err)
	}

	return nil
}

func (a *Auth) Check(inst *Beatport) error {
	currentTime := time.Now().Unix()
	a.mutex.RLock()
	tokenExpirationTime := a.tokenPair.IssuedAt + a.tokenPair.ExpiresIn
	a.mutex.RUnlock()
	if currentTime+300 >= tokenExpirationTime {
		a.mutex.Lock()
		fmt.Println("Refreshing token")
		if _, err := a.refresh(inst); err != nil {
			if err = a.Init(inst); err != nil {
				a.mutex.Unlock()
				return fmt.Errorf("invalid token and authorization error: %w", err)
			}
		}
		a.mutex.Unlock()
	}
	return nil
}

func (a *Auth) Invalidate() {
	a.mutex.Lock()
	a.tokenPair.IssuedAt = 0
	a.mutex.Unlock()
}

func (a *Auth) Init(inst *Beatport) error {
	fmt.Println("Logging in")
	sessionId, err := a.login(inst)
	if err != nil {
		return fmt.Errorf("login: %v", err)
	}
	authorizationCode, err := a.authorize(inst, sessionId)
	if err != nil {
		return fmt.Errorf("authorize: %v", err)
	}
	if err := a.issue(inst, authorizationCode); err != nil {
		return fmt.Errorf("issue token: %v", err)
	}
	return nil
}

func (a *Auth) refresh(inst *Beatport) (*tokenPair, error) {
	payload := map[string]string{
		"client_id":     clientId,
		"refresh_token": a.tokenPair.RefreshToken,
		"grant_type":    "refresh_token",
	}

	res, err := inst.fetch("POST", tokenEndpoint, payload, "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &tokenPair{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	loginId := a.tokenPair.LoginID
	a.tokenPair = response
	a.tokenPair.IssuedAt = time.Now().Unix()
	a.tokenPair.LoginID = loginId
	if err = a.WriteCache(); err != nil {
		return nil, err
	}

	return response, nil
}

func (a *Auth) issue(inst *Beatport, code string) error {
	payload := map[string]string{
		"client_id": clientId,
	}

	if code != "" {
		payload["grant_type"] = "authorization_code"
		payload["code"] = code
	} else {
		payload["grant_type"] = "password"
		payload["username"] = a.username
		payload["password"] = a.password
	}

	res, err := inst.fetch("POST", tokenEndpoint, payload, "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	response := &tokenPair{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return err
	}
	a.tokenPair = response
	a.tokenPair.IssuedAt = time.Now().Unix()
	a.tokenPair.LoginID = a.loginId()
	err = a.WriteCache()
	if err != nil {
		return err
	}

	return nil
}

func (a *Auth) authorize(inst *Beatport, sessionId string) (string, error) {
	inst.headers["cookie"] = fmt.Sprintf("sessionid=%s", sessionId)
	res, err := inst.fetch("GET", authEndpoint, nil, "")
	delete(inst.headers, "cookie")
	if err != nil {
		return "", err
	}

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

func (a *Auth) login(inst *Beatport) (string, error) {
	payload := map[string]string{
		"username": a.username,
		"password": a.password,
	}

	res, err := inst.fetch("POST", loginEndpoint, payload, "application/json")
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

func (a *Auth) loginId() string {
	hash := fnv.New64a()
	data := fmt.Sprintf("%s:%s", a.username, a.password)
	hash.Write([]byte(data))
	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes)
}
