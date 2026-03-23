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
	"strings"
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

type HTTPClientFactory struct{}

func NewHTTPClientFactory() *HTTPClientFactory {
	return &HTTPClientFactory{}
}

func (f *HTTPClientFactory) New(baseURL string) Client {
	return NewHTTPClient(baseURL)
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
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
		return fmt.Errorf("navidrome login failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var ignored map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&ignored); err != nil {
		return fmt.Errorf("decode login response: %w", err)
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
	payload := map[string]any{
		"trackID":  trackID,
		"position": position,
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
	var resp struct {
		Items []playlistInfo `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/playlist", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
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
