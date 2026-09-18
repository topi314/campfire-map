package campfire

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/topi314/campfire-export/server/poi"
)

//go:embed queries/map_objects_by_s2_cells.graphql
var queryMapByS2Cells string

var pgoDropTypesAuthed = []string{
	"PGO_GYM",
	"PGO_POWERSPOT",
	"PGO_POKESTOP",
	"PGO_ROUTE",
}

var pgoDropTypesPublic = []string{
	"PGO_GYM",
	"PGO_POKESTOP",
	"PGO_ROUTE",
}

// DropTypesForToken returns Campfire drop types. Powerspots require a bearer token.
func DropTypesForToken(token string) []string {
	if strings.TrimSpace(token) != "" {
		return append([]string(nil), pgoDropTypesAuthed...)
	}
	return append([]string(nil), pgoDropTypesPublic...)
}

// DropTypesWithoutPowerspot returns gym/stop/route types (no PGO_POWERSPOT).
func DropTypesWithoutPowerspot() []string {
	return append([]string(nil), pgoDropTypesPublic...)
}

// CacheKeySuffix distinguishes authed vs public tile caches.
func CacheKeySuffix(token string) string {
	if strings.TrimSpace(token) != "" {
		return "authed"
	}
	return "public"
}

type Client struct {
	url              string
	http             *http.Client
	retries          int
	realityChannelID string
	s2CellLevel      int
}

func New(cfg Config) *Client {
	if cfg.URL == "" {
		cfg.URL = "https://niantic-social-api.nianticlabs.com/graphql"
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RealityChannelID == "" {
		cfg.RealityChannelID = DefaultRealityChannelID
	}
	if cfg.S2CellLevel <= 0 {
		cfg.S2CellLevel = 15
	}
	return &Client{
		url:              cfg.URL,
		http:             &http.Client{Timeout: 25 * time.Second},
		retries:          cfg.MaxRetries,
		realityChannelID: cfg.RealityChannelID,
		s2CellLevel:      cfg.S2CellLevel,
	}
}

type gqlReq struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type gqlErr struct {
	Message string `json:"message"`
}

type gqlResp struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlErr        `json:"errors"`
}

func (c *Client) Fetch(ctx context.Context, cellIDs []string, level int) ([]poi.POI, error) {
	return c.FetchWithDropTypes(ctx, cellIDs, level, DropTypesForToken(tokenFor(ctx)))
}

// FetchWithDropTypes loads map objects using the given PGO drop types.
func (c *Client) FetchWithDropTypes(ctx context.Context, cellIDs []string, level int, dropTypes []string) ([]poi.POI, error) {
	// Campfire returns full POI details at S2 level 15.
	level = 15
	if len(dropTypes) == 0 {
		dropTypes = DropTypesForToken(tokenFor(ctx))
	}
	raw, err := c.do(ctx, queryMapByS2Cells, c.mapByS2CellsVars(cellIDs, level, dropTypes))
	if err != nil {
		return nil, err
	}
	return parseMapObjects(raw)
}

func (c *Client) mapByS2CellsVars(cellIDs []string, level int, dropTypes []string) map[string]any {
	sourcesByS2Cells := make([]map[string]any, len(cellIDs))
	for i, id := range cellIDs {
		sourcesByS2Cells[i] = map[string]any{
			"s2CellId": id,
			"sources": []map[string]any{
				{
					"name":      "PGO",
					"dropTypes": dropTypes,
				},
			},
		}
	}
	return map[string]any{
		"realityChannelMapObjectsByS2CellsInput": map[string]any{
			"realityChannelId": c.realityChannelID,
			"s2CellLevel":      level,
			"sourcesByS2Cells": sourcesByS2Cells,
		},
	}
}

func (c *Client) do(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	var last error
	for attempt := 0; attempt < c.retries; attempt++ {
		raw, err := c.roundTrip(ctx, query, vars)
		if err != nil {
			last = err
			if isRetryable(err) {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return nil, err
		}
		return raw, nil
	}
	if last == nil {
		last = fmt.Errorf("too many retries")
	}
	return nil, last
}

func (c *Client) roundTrip(ctx context.Context, query string, vars map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(gqlReq{Query: query, Variables: vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "campfire-export/1.0")
	if name := operationName(query); name != "" {
		req.Header.Set("X-APOLLO-OPERATION-NAME", name)
	}
	if tok := tokenFor(ctx); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql http %s: %s", resp.Status, truncate(data, 400))
	}

	var parsed gqlResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("decode graphql: %w", err)
	}
	if len(parsed.Errors) > 0 {
		msgs := make([]string, 0, len(parsed.Errors))
		for _, e := range parsed.Errors {
			msgs = append(msgs, e.Message)
		}
		joined := strings.Join(msgs, "; ")
		if strings.Contains(joined, "Cannot query field") || strings.Contains(joined, "Unknown argument") {
			return nil, fmt.Errorf("schema mismatch: %s", joined)
		}
		if parsed.Data == nil || bytes.Equal(parsed.Data, []byte("null")) {
			return nil, fmt.Errorf("graphql: %s", joined)
		}
		slog.WarnContext(ctx, "graphql partial errors", slog.String("errors", joined))
	}
	if parsed.Data == nil || bytes.Equal(parsed.Data, []byte("null")) {
		return nil, fmt.Errorf("empty graphql data")
	}
	return parsed.Data, nil
}

type tokenCtxKey struct{}

// WithToken attaches a per-request Campfire/Pokémon GO bearer token.
func WithToken(ctx context.Context, token string) context.Context {
	token = strings.TrimSpace(token)
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenCtxKey{}, token)
}

func tokenFor(ctx context.Context) string {
	t, _ := ctx.Value(tokenCtxKey{}).(string)
	return t
}

func operationName(query string) string {
	for _, line := range strings.Split(query, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "query ") || strings.HasPrefix(line, "mutation ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name := strings.TrimRight(fields[1], "({")
				if name != "" {
					return name
				}
			}
		}
	}
	return ""
}

func isRetryable(err error) bool {
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(s, "502") || strings.Contains(s, "DeadlineExceeded")
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
