package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type placeResult struct {
	ID    int64   `json:"id"`
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	South float64 `json:"south"`
	North float64 `json:"north"`
	West  float64 `json:"west"`
	East  float64 `json:"east"`
}

type nominatimRow struct {
	PlaceID     int64    `json:"place_id"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	DisplayName string   `json:"display_name"`
	BoundingBox []string `json:"boundingbox"`
}

var placesHTTP = &http.Client{Timeout: 12 * time.Second}

func (s *Server) searchPlaces(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"places": []placeResult{}})
		return
	}
	if len(q) > 200 {
		http.Error(w, "query too long", http.StatusBadRequest)
		return
	}

	params := url.Values{}
	params.Set("format", "json")
	params.Set("q", q)
	params.Set("limit", "8")
	params.Set("addressdetails", "0")
	params.Set("dedupe", "1")

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://nominatim.openstreetmap.org/search?"+params.Encode(), nil)
	if err != nil {
		http.Error(w, "place search failed", http.StatusBadGateway)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "de,en")
	req.Header.Set("User-Agent", "campfire-map-export/1.0 (local map tool)")

	resp, err := placesHTTP.Do(req)
	if err != nil {
		http.Error(w, "place search failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		http.Error(w, "place search failed", http.StatusBadGateway)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("place search failed (%d)", resp.StatusCode), http.StatusBadGateway)
		return
	}

	var rows []nominatimRow
	if err := json.Unmarshal(body, &rows); err != nil {
		http.Error(w, "place search failed", http.StatusBadGateway)
		return
	}

	places := make([]placeResult, 0, len(rows))
	for _, row := range rows {
		lat, err1 := strconv.ParseFloat(row.Lat, 64)
		lng, err2 := strconv.ParseFloat(row.Lon, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		p := placeResult{
			ID:    row.PlaceID,
			Label: row.DisplayName,
			Lat:   lat,
			Lng:   lng,
			South: lat,
			North: lat,
			West:  lng,
			East:  lng,
		}
		if len(row.BoundingBox) == 4 {
			if south, err := strconv.ParseFloat(row.BoundingBox[0], 64); err == nil {
				p.South = south
			}
			if north, err := strconv.ParseFloat(row.BoundingBox[1], 64); err == nil {
				p.North = north
			}
			if west, err := strconv.ParseFloat(row.BoundingBox[2], 64); err == nil {
				p.West = west
			}
			if east, err := strconv.ParseFloat(row.BoundingBox[3], 64); err == nil {
				p.East = east
			}
		}
		places = append(places, p)
	}

	writeJSON(w, http.StatusOK, map[string]any{"places": places})
}
