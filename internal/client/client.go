package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/veeamgo/veeamgo/internal/config"
	"github.com/veeamgo/veeamgo/internal/session"
)

// Credentials represents login parameters.
type Credentials struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	Scope    string
}

// ErrRefreshTokenMissing indicates that the session does not contain a refresh token.
var ErrRefreshTokenMissing = errors.New("refresh token missing")

// tokenResponse models the Veeam OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// Client wraps HTTP operations against the Veeam API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	session    *session.Session
}

// APIError captures structured error responses returned by the Veeam REST API.
type APIError struct {
	StatusCode int
	ErrorCode  string
	Title      string
	Message    string
	Raw        string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	title := strings.TrimSpace(e.Title)
	message := strings.TrimSpace(e.Message)

	switch {
	case title != "" && message != "":
		if strings.EqualFold(title, message) {
			return message
		}
		return fmt.Sprintf("%s: %s", title, message)
	case message != "":
		return message
	case title != "":
		return title
	case e.Raw != "":
		return e.Raw
	default:
		return fmt.Sprintf("request failed with status %d", e.StatusCode)
	}
}

// Authenticate exchanges credentials for access and refresh tokens.
func Authenticate(ctx context.Context, creds Credentials) (*session.Session, error) {
	if creds.BaseURL == "" {
		return nil, errors.New("server URL is required")
	}
	if creds.Username == "" {
		return nil, errors.New("username is required")
	}
	if creds.Password == "" {
		return nil, errors.New("password is required")
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", creds.Username)
	form.Set("password", creds.Password)
	if creds.Scope != "" {
		form.Set("scope", creds.Scope)
	}

	return requestToken(ctx, creds.BaseURL, creds.Insecure, form)
}

// Refresh exchanges a refresh token for a new access token set.
func Refresh(ctx context.Context, profile *config.Profile, sess *session.Session) (*session.Session, error) {
	if sess == nil {
		return nil, errors.New("session is required")
	}
	if profile == nil {
		return nil, errors.New("profile is required")
	}
	if sess.RefreshToken == "" {
		return nil, ErrRefreshTokenMissing
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", sess.RefreshToken)

	return requestToken(ctx, profile.ServerURL, profile.Insecure, form)
}

// New returns an authenticated client for the given profile and session.
func New(profile *config.Profile, sess *session.Session) (*Client, error) {
	if profile == nil {
		return nil, errors.New("profile is required")
	}
	if sess == nil {
		return nil, errors.New("session is required")
	}
	base, err := parseBaseURL(profile.ServerURL)
	if err != nil {
		return nil, err
	}
	httpClient := newHTTPClient(profile.Insecure)
	return &Client{
		baseURL:    base,
		httpClient: httpClient,
		session:    sess,
	}, nil
}

// ServerInfo fetches metadata about the VBR server.
type ServerInfo struct {
	Name             string   `json:"name"`
	Platform         string   `json:"platform"`
	VBRID            string   `json:"vbrId"`
	BuildVersion     string   `json:"buildVersion"`
	DatabaseVendor   string   `json:"databaseVendor"`
	DatabaseContent  string   `json:"databaseContentVersion"`
	DatabaseSchema   string   `json:"databaseSchemaVersion"`
	DatabaseEdition  string   `json:"databaseEdition"`
	SQLServerEdition string   `json:"sqlServerEdition"`
	SQLServerVersion string   `json:"sqlServerVersion"`
	Patches          []string `json:"patches"`
}

func (c *Client) ServerInfo(ctx context.Context) (*ServerInfo, error) {
	var payload ServerInfo
	if err := c.getJSON(ctx, "/api/v1/serverInfo", &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ServerTime represents the server clock payload.
type ServerTime struct {
	Time                time.Time `json:"serverTime"`
	TimeZone            string    `json:"timeZone"`
	TimeZoneDisplayName string    `json:"timezoneDisplayName"`
	TimeZoneID          string    `json:"timezoneId"`
	UtcOffsetMinutes    int       `json:"utcOffsetMinutes"`
}

// ServerTime fetches the current server time.
func (c *Client) ServerTime(ctx context.Context) (*ServerTime, error) {
	var payload ServerTime
	if err := c.getJSON(ctx, "/api/v1/serverTime", &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func requestToken(ctx context.Context, baseURL string, insecure bool, form url.Values) (*session.Session, error) {
	base, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.ResolveReference(&url.URL{Path: "/api/oauth2/token"}).String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	httpClient := newHTTPClient(insecure)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("token request failed: %s", strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, errors.New("token response missing access token")
	}

	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	sess := &session.Session{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    normalizeTokenType(token.TokenType),
		ExpiresAt:    expiresAt,
	}
	return sess, nil
}

func normalizeTokenType(tokenType string) string {
	if tokenType == "" {
		return "bearer"
	}
	return strings.ToLower(tokenType)
}

func (c *Client) getJSON(ctx context.Context, path string, target any) error {
	return c.getJSONWithQuery(ctx, path, nil, target)
}

func (c *Client) getJSONWithQuery(ctx context.Context, path string, query url.Values, target any) error {
	rel := &url.URL{Path: path}
	if query != nil && len(query) > 0 {
		rel.RawQuery = query.Encode()
	}
	return c.doRequest(ctx, http.MethodGet, rel, nil, "", target)
}

func (c *Client) postJSON(ctx context.Context, path string, query url.Values, payload any, target any) error {
	rel := &url.URL{Path: path}
	if query != nil && len(query) > 0 {
		rel.RawQuery = query.Encode()
	}

	var body io.Reader
	contentType := ""
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = buf
		contentType = "application/json"
	}

	return c.doRequest(ctx, http.MethodPost, rel, body, contentType, target)
}

func (c *Client) putJSON(ctx context.Context, path string, query url.Values, payload any, target any) error {
	rel := &url.URL{Path: path}
	if query != nil && len(query) > 0 {
		rel.RawQuery = query.Encode()
	}

	var body io.Reader
	contentType := ""
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = buf
		contentType = "application/json"
	}

	return c.doRequest(ctx, http.MethodPut, rel, body, contentType, target)
}

func (c *Client) delete(ctx context.Context, path string, query url.Values) error {
	rel := &url.URL{Path: path}
	if query != nil && len(query) > 0 {
		rel.RawQuery = query.Encode()
	}
	return c.doRequest(ctx, http.MethodDelete, rel, nil, "", nil)
}

func (c *Client) doRequest(ctx context.Context, method string, rel *url.URL, body io.Reader, contentType string, target any) error {
	reqURL := c.baseURL.ResolveReference(rel)
	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.session.TokenType != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.session.TokenType, c.session.AccessToken))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.session.AccessToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return parseAPIError(resp.StatusCode, bodyBytes)
	}

	if target == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("server URL is empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse server url: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u, nil
}

func newHTTPClient(insecure bool) *http.Client {
	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // Explicit user opt-in.
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
}

func parseAPIError(status int, body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	apiErr := &APIError{
		StatusCode: status,
		Raw:        trimmed,
	}

	if len(trimmed) == 0 {
		return apiErr
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr.ErrorCode = stringFromAny(payload["errorCode"])
		apiErr.Title = stringFromAny(payload["title"])
		apiErr.Message = stringFromAny(payload["message"])

		if apiErr.Message == "" {
			apiErr.Message = stringFromAny(payload["detail"])
		}
		if apiErr.Message == "" {
			apiErr.Message = stringFromAny(payload["error"])
		}
		if apiErr.Message == "" {
			apiErr.Message = stringFromAny(payload["localizedMessage"])
		}
		if apiErr.Message == "" {
			apiErr.Message = extractDetailMessage(payload["details"])
		}
		if apiErr.Message == "" {
			apiErr.Message = extractDetailMessage(payload["errors"])
		}
		if apiErr.Title == "" {
			apiErr.Title = stringFromAny(payload["error"])
		}
		if apiErr.Title == "" {
			apiErr.Title = stringFromAny(payload["status"])
		}
	}

	if apiErr.Message == "" && apiErr.Title == "" && apiErr.Raw == "" {
		apiErr.Raw = fmt.Sprintf("request failed with status %d", status)
	}
	return apiErr
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case float64, float32, int, int32, int64, uint, uint32, uint64:
		return strings.TrimSpace(fmt.Sprint(v))
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func extractDetailMessage(value any) string {
	switch typed := value.(type) {
	case []any:
		messages := make([]string, 0, len(typed))
		for _, item := range typed {
			if msg := detailMessageFromItem(item); msg != "" {
				messages = append(messages, msg)
			}
		}
		return strings.Join(messages, "; ")
	case map[string]any:
		return detailMessageFromItem(typed)
	default:
		return strings.TrimSpace(stringFromAny(value))
	}
}

func detailMessageFromItem(item any) string {
	switch typed := item.(type) {
	case map[string]any:
		if msg := stringFromAny(typed["message"]); msg != "" {
			return msg
		}
		if msg := stringFromAny(typed["localizedMessage"]); msg != "" {
			return msg
		}
		if msg := stringFromAny(typed["detail"]); msg != "" {
			return msg
		}
		return strings.TrimSpace(fmt.Sprint(typed))
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
