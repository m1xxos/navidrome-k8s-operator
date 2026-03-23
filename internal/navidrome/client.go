package navidrome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TrackSelector struct {
	TrackID  string
	FilePath string
	Artist   string
	Title    string
}

type Client interface {
	Login(ctx context.Context, username, password string) error
	EnsurePlaylist(ctx context.Context, name string) (string, error)
	DeletePlaylist(ctx context.Context, playlistID string) error
	ResolveTrack(ctx context.Context, selector TrackSelector) (string, error)
	AddOrMoveTrack(ctx context.Context, playlistID, trackID string, position int) error
	RemoveTrack(ctx context.Context, playlistID, trackID string) error
}

type ClientFactory interface {
	New(baseURL string) Client
}

type HTTPClientFactory struct {
	mu      sync.Mutex
	clients map[string]*HTTPClient
}

func NewHTTPClientFactory() *HTTPClientFactory {
	return &HTTPClientFactory{clients: map[string]*HTTPClient{}}
}

func (f *HTTPClientFactory) New(baseURL string) Client {
	normalized := strings.TrimRight(baseURL, "/")

	f.mu.Lock()
	defer f.mu.Unlock()

	if existing, ok := f.clients[normalized]; ok {
		return existing
	}

	created := NewHTTPClient(normalized)
	f.clients[normalized] = created
	return created
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client

	mu      sync.RWMutex
	token   string
	authed  bool
	loginMu sync.Mutex
}

func NewHTTPClient(baseURL string) *HTTPClient {
	jar, _ := cookiejar.New(nil)
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
	}
}

func (c *HTTPClient) Login(ctx context.Context, username, password string) error {
	if c.isAuthenticated() {
		return nil
	}

	c.loginMu.Lock()
	defer c.loginMu.Unlock()

	if c.isAuthenticated() {
		return nil
	}

	const maxAttempts = 4
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := c.loginOnce(ctx, username, password)
		if err == nil {
			return nil
		}

		if !isRateLimitedLoginError(err) || attempt == maxAttempts {
			return err
		}

		delay := backoff
		if retryAfter, ok := retryAfterFromError(err); ok && retryAfter > delay {
			delay = retryAfter
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		if backoff < 4*time.Second {
			backoff *= 2
		}
	}

	return errors.New("navidrome login failed after retries")
}

func (c *HTTPClient) loginOnce(ctx context.Context, username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := fmt.Sprintf("navidrome login failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter > 0 {
				return fmt.Errorf("%s (retryAfter=%s)", msg, retryAfter)
			}
		}
		return errors.New(msg)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}

	if token := pickToken(result); token != "" {
		c.mu.Lock()
		c.token = token
		c.authed = true
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.authed = true
		c.mu.Unlock()
	}

	return nil
}

func (c *HTTPClient) EnsurePlaylist(ctx context.Context, name string) (string, error) {
	playlists, err := c.listPlaylists(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range playlists {
		if strings.EqualFold(p.Name, name) {
			return p.ID, nil
		}
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/playlist", map[string]any{"name": name}, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", errors.New("navidrome create playlist returned empty id")
	}
	return created.ID, nil
}

func (c *HTTPClient) DeletePlaylist(ctx context.Context, playlistID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/playlist/"+url.PathEscape(playlistID), nil, nil)
}

func (c *HTTPClient) ResolveTrack(ctx context.Context, selector TrackSelector) (string, error) {
	if selector.TrackID != "" {
		return selector.TrackID, nil
	}

	query := url.Values{}
	if selector.FilePath != "" {
		query.Set("path", selector.FilePath)
	}
	if selector.Artist != "" {
		query.Set("artist", selector.Artist)
	}
	if selector.Title != "" {
		query.Set("title", selector.Title)
	}

	endpoint := "/api/song"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 || resp.Items[0].ID == "" {
		return "", errors.New("track not found in navidrome")
	}
	return resp.Items[0].ID, nil
}

func (c *HTTPClient) AddOrMoveTrack(ctx context.Context, playlistID, trackID string, position int) error {
	_ = position
	payload := map[string]any{
		"ids": []string{trackID},
	}
	endpoint := fmt.Sprintf("/api/playlist/%s/tracks", url.PathEscape(playlistID))
	return c.doJSON(ctx, http.MethodPost, endpoint, payload, nil)
}

func (c *HTTPClient) RemoveTrack(ctx context.Context, playlistID, trackID string) error {
	endpoint := fmt.Sprintf("/api/playlist/%s/tracks/%s", url.PathEscape(playlistID), url.PathEscape(trackID))
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

type playlistInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *HTTPClient) listPlaylists(ctx context.Context) ([]playlistInfo, error) {
	var raw any
	if err := c.doJSON(ctx, http.MethodGet, "/api/playlist", nil, &raw); err != nil {
		return nil, err
	}

	if arr, ok := raw.([]any); ok {
		return parsePlaylistArray(arr), nil
	}

	if obj, ok := raw.(map[string]any); ok {
		if arr, ok := obj["items"].([]any); ok {
			return parsePlaylistArray(arr), nil
		}
	}

	return nil, fmt.Errorf("unexpected playlist response format")
}

func parsePlaylistArray(arr []any) []playlistInfo {
	out := make([]playlistInfo, 0, len(arr))
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := obj["id"].(string)
		name, _ := obj["name"].(string)
		if id == "" {
			continue
		}
		out = append(out, playlistInfo{ID: id, Name: name})
	}
	return out
}

func (c *HTTPClient) doJSON(ctx context.Context, method, endpoint string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()
	if token != "" {
		bearer := "Bearer " + token
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-ND-Authorization", bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			c.mu.Lock()
			c.token = ""
			c.authed = false
			c.mu.Unlock()
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("navidrome api %s %s failed: %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func (c *HTTPClient) isAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != "" || c.authed
}

func isRateLimitedLoginError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "status 429")
}

func retryAfterFromError(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	const marker = "retryAfter="
	msg := err.Error()
	idx := strings.LastIndex(msg, marker)
	if idx < 0 {
		return 0, false
	}
	raw := strings.TrimSuffix(msg[idx+len(marker):], ")")
	d, parseErr := time.ParseDuration(raw)
	if parseErr != nil {
		return 0, false
	}
	return d, true
}

func parseRetryAfter(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}

	if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}

	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}

func pickToken(m map[string]any) string {
	keys := []string{"token", "idToken", "jwt", "accessToken"}
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			if s, ok := raw.(string); ok {
				return s
			}
		}
	}
	return ""
}
