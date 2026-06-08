package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/formation-res/open-location-hub-cli/internal/openapi"
)

type Config struct {
	BaseURL string
	Token   string
	JSON    bool
	NoColor bool
	Timeout time.Duration
	EnvFile string
	OAuth   OAuthConfig
}

type OAuthConfig struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Scope        string
	GrantType    string
	Audience     string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

func (c Config) HTTPClient() *http.Client {
	return &http.Client{Timeout: c.Timeout}
}

func (c Config) APIClient() (*openapi.ClientWithResponses, error) {
	trimmed := strings.TrimRight(c.BaseURL, "/")
	if trimmed == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	cli, err := openapi.NewClientWithResponses(trimmed, openapi.WithHTTPClient(c.HTTPClient()), openapi.WithRequestEditorFn(c.requestEditor))
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (c Config) requestEditor(_ context.Context, req *http.Request) error {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Accept", "application/json")
	return nil
}

func EnvOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func DefaultEnvFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".openlocationhub.env")
}

func LoadEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}
	if strings.TrimSpace(path) == "" {
		return values, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		values[key] = val
	}
	return values, nil
}

func ResolveValue(env map[string]string, key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	if v := strings.TrimSpace(env[key]); v != "" {
		return v
	}
	return fallback
}

func (c *Config) EnsureToken(ctx context.Context) error {
	if strings.TrimSpace(c.Token) != "" {
		return nil
	}
	if strings.TrimSpace(c.OAuth.TokenURL) == "" {
		return nil
	}
	resp, err := c.FetchToken(ctx)
	if err != nil {
		return err
	}
	c.Token = resp.AccessToken
	return nil
}

func (c Config) FetchToken(ctx context.Context) (*TokenResponse, error) {
	if strings.TrimSpace(c.OAuth.TokenURL) == "" {
		return nil, fmt.Errorf("oauth token URL is not configured")
	}
	form := url.Values{}
	grantType := c.OAuth.GrantType
	if grantType == "" {
		grantType = "password"
	}
	form.Set("grant_type", grantType)
	if c.OAuth.Scope != "" {
		form.Set("scope", c.OAuth.Scope)
	}
	if c.OAuth.Audience != "" {
		form.Set("audience", c.OAuth.Audience)
	}
	switch grantType {
	case "password":
		form.Set("username", c.OAuth.Username)
		form.Set("password", c.OAuth.Password)
	case "client_credentials":
	default:
		return nil, fmt.Errorf("unsupported oauth grant type %q", grantType)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.OAuth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.OAuth.ClientID != "" || c.OAuth.ClientSecret != "" {
		req.SetBasicAuth(c.OAuth.ClientID, c.OAuth.ClientSecret)
	}
	res, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("token endpoint returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint response did not contain access_token")
	}
	return &tokenResp, nil
}

func WriteEnvFile(path string, values map[string]string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, values[key]))
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
