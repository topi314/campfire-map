package campfire

import (
	"encoding/json"
	"strings"
)

// Response shapes for queries/map_objects_by_s2_cells.graphql.

type tileResponse struct {
	RealityChannelMapObjectsByS2Cells *struct {
		MapObjectsByS2CellsAndTypes []struct {
			S2CellID         string             `json:"s2CellId"`
			MapObjectsByType []mapObjectsByType `json:"mapObjectsByType"`
		} `json:"mapObjectsByS2CellsAndTypes"`
	} `json:"realityChannelMapObjectsByS2Cells"`
}

type mapObjectsByType struct {
	Type       string      `json:"type"`
	MapObjects []mapObject `json:"mapObjects"`
}

type mapObject struct {
	MapObjectType string        `json:"mapObjectType"`
	PgoGym        *pgoGym       `json:"pgoGym"`
	PgoPokestop   *pgoPokestop  `json:"pgoPokestop"`
	PgoPowerspot  *pgoPowerspot `json:"pgoPowerspot"`
	PgoRoute      *pgoRoute     `json:"pgoRoute"`
}

type latLng struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (l *latLng) ok() (lat, lng float64, ok bool) {
	if l == nil || l.Latitude == nil || l.Longitude == nil {
		return 0, 0, false
	}
	return *l.Latitude, *l.Longitude, true
}

type pgoGym struct {
	Name                   string       `json:"name"`
	ImageURL               string       `json:"imageUrl"`
	IsMegaEnhancedEligible flexibleJSON `json:"isMegaEnhancedEligible"`
	Location               *latLng      `json:"location"`
}

type pgoPokestop struct {
	Name     string  `json:"name"`
	ImageURL string  `json:"imageUrl"`
	Location *latLng `json:"location"`
}

type pgoPowerspot struct {
	Name     string  `json:"name"`
	Location *latLng `json:"location"`
}

type pgoRoute struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	DistanceMeters  *float64  `json:"distanceMeters"`
	DurationSeconds *float64  `json:"durationSeconds"`
	Reversible      *bool     `json:"reversible"`
	StartPoi        *routePoi `json:"startPoi"`
	EndPoi          *routePoi `json:"endPoi"`
	LocationList    []latLng  `json:"locationList"`
}

type routePoi struct {
	FortID   string  `json:"fortId"`
	ImageURL string  `json:"imageUrl"`
	Location *latLng `json:"location"`
}

// flexibleJSON accepts bool/number/string/object values from Campfire.
type flexibleJSON struct {
	raw json.RawMessage
}

func (f *flexibleJSON) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		f.raw = nil
		return nil
	}
	f.raw = append(json.RawMessage(nil), b...)
	return nil
}

func (f flexibleJSON) empty() bool {
	return len(f.raw) == 0 || string(f.raw) == "null"
}

func (f flexibleJSON) truthy() bool {
	if f.empty() {
		return false
	}
	var b bool
	if err := json.Unmarshal(f.raw, &b); err == nil {
		return b
	}
	var n float64
	if err := json.Unmarshal(f.raw, &n); err == nil {
		return n != 0
	}
	var s string
	if err := json.Unmarshal(f.raw, &s); err == nil {
		return truthyString(s)
	}
	var m map[string]any
	if err := json.Unmarshal(f.raw, &m); err == nil {
		for _, k := range []string{"eligible", "isEligible", "value", "enabled"} {
			if v, ok := m[k]; ok && truthyAny(v) {
				return true
			}
		}
	}
	return false
}

func truthyString(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "false" || s == "0" || s == "no" ||
		strings.Contains(s, "ineligible") || strings.Contains(s, "not_eligible") {
		return false
	}
	return s == "true" || s == "1" || s == "yes" || s == "eligible" ||
		s == "mega_enhanced" || s == "super_mega" || strings.Contains(s, "eligible")
}

func truthyAny(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return truthyString(t)
	default:
		return false
	}
}
