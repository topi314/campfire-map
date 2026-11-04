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

var pgoDropTypesPublic = []string{
	"PGO_GYM",
	"PGO_POKESTOP",
	"PGO_ROUTE",
}

// DropTypesWithoutPowerspot returns gym/stop/route types (no PGO_POWERSPOT).
// Powerspots are loaded from Wayfarer instead.
func DropTypesWithoutPowerspot() []string {
	return append([]string(nil), pgoDropTypesPublic...)
}

// CacheKeySuffix is the shared Campfire tile cache key suffix (public GraphQL only).
func CacheKeySuffix() string {
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
	return c.FetchWithDropTypes(ctx, cellIDs, level, DropTypesWithoutPowerspot())
}

// FetchWithDropTypes loads map objects using the given PGO drop types.
func (c *Client) FetchWithDropTypes(ctx context.Context, cellIDs []string, level int, dropTypes []string) ([]poi.POI, error) {
	// Campfire returns full POI details at S2 level 15.
	level = 15
	if len(dropTypes) == 0 {
		dropTypes = DropTypesWithoutPowerspot()
	}
	if len(cellIDs) == 0 {
		return nil, nil
	}
	var lists [][]poi.POI
	var lastErr error
	for _, batch := range chunkStrings(cellIDs, cellBatchSize) {
		pois, err := c.fetchBatch(ctx, batch, level, dropTypes)
		if err != nil {
			lastErr = err
			slog.WarnContext(ctx, "campfire cell batch", slog.Int("cells", len(batch)), slog.Any("err", err))
			continue
		}
		lists = append(lists, pois)
	}
	if len(lists) == 0 {
		return nil, lastErr
	}
	return mergePOILists(lists...), nil
}

// cellBatchSize keeps each Campfire platform call small. Larger requests sometimes
// come back as GET_MAP_OBJECTS_FOR_CAMPFIRE HTTP 400 and fail the whole viewport.
const cellBatchSize = 64

func (c *Client) fetchOnce(ctx context.Context, cellIDs []string, level int, dropTypes []string) ([]poi.POI, error) {
	raw, err := c.do(ctx, queryMapByS2Cells, c.mapByS2CellsVars(cellIDs, level, dropTypes))
	if err != nil {
		return nil, err
	}
	return parseMapObjects(raw)
}

// fetchBatch loads cells, retrying a platform 400 once and then splitting the batch.
// A single rejected cell is skipped so the rest of the viewport still loads.
func (c *Client) fetchBatch(ctx context.Context, cellIDs []string, level int, dropTypes []string) ([]poi.POI, error) {
	pois, err := c.fetchOnce(ctx, cellIDs, level, dropTypes)
	if err == nil || !isPlatformBadRequest(err) {
		return pois, err
	}
	if len(cellIDs) < 2 {
		return nil, err
	}
	if retried, err2 := c.fetchOnce(ctx, cellIDs, level, dropTypes); err2 == nil {
		return retried, nil
	}
	mid := len(cellIDs) / 2
	left, errL := c.fetchBatch(ctx, cellIDs[:mid], level, dropTypes)
	right, errR := c.fetchBatch(ctx, cellIDs[mid:], level, dropTypes)
	if errL != nil && errR != nil {
		return nil, err
	}
	if errL != nil {
		slog.WarnContext(ctx, "skipping campfire cells", slog.Int("cells", mid), slog.Any("err", errL))
		return right, nil
	}
	if errR != nil {
		slog.WarnContext(ctx, "skipping campfire cells", slog.Int("cells", len(cellIDs)-mid), slog.Any("err", errR))
		return left, nil
	}
	return mergePOILists(left, right), nil
}

func isPlatformBadRequest(err error) bool {
	return err != nil && strings.Contains(err.Error(), "HTTP Mapping: 400")
}

func chunkStrings(ids []string, size int) [][]string {
	if size < 1 {
		size = 1
	}
	out := make([][]string, 0, (len(ids)+size-1)/size)
	for i := 0; i < len(ids); i += size {
		j := i + size
		if j > len(ids) {
			j = len(ids)
		}
		out = append(out, ids[i:j])
	}
	return out
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
