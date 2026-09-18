package wayfarer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/topi314/campfire-export/server/poi"
)

const cellLevel = 14

type Client struct {
	url     string
	http    *http.Client
	retries int
}

func New(cfg Config) *Client {
	if cfg.URL == "" {
		cfg.URL = DefaultURL
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	return &Client{
		url:     cfg.URL,
		http:    &http.Client{Timeout: 25 * time.Second},
		retries: cfg.MaxRetries,
	}
}

// Creds are per-request Wayfarer session cookies from the browser.
type Creds struct {
	Session string
	XSRF    string
}

func (c Creds) OK() bool {
	return strings.TrimSpace(c.Session) != "" && strings.TrimSpace(c.XSRF) != ""
}

type AuthError struct {
	Status int
	Body   string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("wayfarer auth failed (%d): %s", e.Status, e.Body)
}

func (c *Client) FetchPowerspots(ctx context.Context, bbox poi.BBox, creds Creds) ([]poi.POI, error) {
	if !creds.OK() {
		return nil, fmt.Errorf("wayfarer session and xsrf token required")
	}
	var last error
	for attempt := 0; attempt < c.retries; attempt++ {
		pois, err := c.roundTrip(ctx, bbox, creds)
		if err != nil {
			last = err
			if isRetryable(err) {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return nil, err
		}
		return pois, nil
	}
	if last == nil {
		last = fmt.Errorf("too many retries")
	}
	return nil, last
}

func (c *Client) roundTrip(ctx context.Context, bbox poi.BBox, creds Creds) ([]poi.POI, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("ne", fmt.Sprintf("(%g,%g)", bbox.MaxLat, bbox.MaxLng))
	q.Set("sw", fmt.Sprintf("(%g,%g)", bbox.MinLat, bbox.MinLng))
	q.Set("cellLevel", fmt.Sprintf("%d", cellLevel))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	session := strings.TrimSpace(creds.Session)
	xsrf := strings.TrimSpace(creds.XSRF)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "campfire-export/1.0")
	req.Header.Set("Cookie", fmt.Sprintf("SESSION=%s; XSRF-TOKEN=%s", session, xsrf))
	req.Header.Set("X-XSRF-TOKEN", xsrf)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusBadGateway {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{Status: resp.StatusCode, Body: truncate(data, 200)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wayfarer http %s: %s", resp.Status, truncate(data, 400))
	}
	return parsePowerspots(data)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(s, "502") || strings.Contains(s, "DeadlineExceeded")
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
