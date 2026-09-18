package wayfarer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/topi314/campfire-export/server/poi"
)

type apiResp struct {
	Result *apiResult `json:"result"`
	Code   string     `json:"code"`
}

type apiResult struct {
	Success bool      `json:"success"`
	Data    []apiCell `json:"data"`
}

type apiCell struct {
	Pois []apiPOI `json:"pois"`
}

type apiPOI struct {
	PoiID string   `json:"poiId"`
	LatE6 int      `json:"latE6"`
	LngE6 int      `json:"lngE6"`
	Title string   `json:"title"`
	GMO   []apiGMO `json:"gmo"`
}

type apiGMO struct {
	GameBrand string `json:"gameBrand"`
	Entity    string `json:"entity"`
	Status    string `json:"status"`
}

func parsePowerspots(data []byte) ([]poi.POI, error) {
	var resp apiResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode wayfarer: %w", err)
	}
	if resp.Result == nil {
		return nil, fmt.Errorf("wayfarer: empty result")
	}
	if !resp.Result.Success && resp.Code != "" && resp.Code != "OK" {
		return nil, fmt.Errorf("wayfarer: %s", resp.Code)
	}

	seen := map[string]struct{}{}
	out := make([]poi.POI, 0)
	for _, cell := range resp.Result.Data {
		for _, p := range cell.Pois {
			spot, ok := p.toPowerspot()
			if !ok {
				continue
			}
			if _, dup := seen[spot.ID]; dup {
				continue
			}
			seen[spot.ID] = struct{}{}
			out = append(out, spot)
		}
	}
	return out, nil
}

func (p apiPOI) toPowerspot() (poi.POI, bool) {
	status, ok := powerspotStatus(p.GMO)
	if !ok {
		return poi.POI{}, false
	}
	id := strings.TrimSpace(p.PoiID)
	if id == "" {
		return poi.POI{}, false
	}
	name := strings.TrimSpace(p.Title)
	if name == "" {
		name = "Powerspot"
	}
	return poi.POI{
		ID:     id,
		Type:   poi.TypePowerspot,
		Name:   name,
		Lat:    float64(p.LatE6) / 1e6,
		Lng:    float64(p.LngE6) / 1e6,
		Status: status,
	}, true
}

func powerspotStatus(gmos []apiGMO) (string, bool) {
	for _, g := range gmos {
		if !strings.EqualFold(g.GameBrand, "HOLOHOLO") {
			continue
		}
		if !strings.EqualFold(g.Entity, "POWERSPOT") {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(g.Status)) {
		case "INACTIVE":
			return poi.StatusInactive, true
		default:
			return poi.StatusActive, true
		}
	}
	return "", false
}
