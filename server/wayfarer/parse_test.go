package wayfarer

import (
	"testing"

	"github.com/topi314/campfire-export/server/poi"
)

func TestParsePowerspots(t *testing.T) {
	raw := []byte(`{
		"result": {
			"success": true,
			"data": [{
				"pois": [
					{
						"poiId": "active-1",
						"latE6": 50108230,
						"lngE6": 8760796,
						"title": "Active Spot",
						"gmo": [{"gameBrand":"HOLOHOLO","entity":"POWERSPOT","status":"ACTIVE"}]
					},
					{
						"poiId": "inactive-1",
						"latE6": 50106375,
						"lngE6": 8759598,
						"title": "Inactive Spot",
						"gmo": [{"gameBrand":"HOLOHOLO","entity":"POWERSPOT","status":"INACTIVE"}]
					},
					{
						"poiId": "stop-1",
						"latE6": 50107067,
						"lngE6": 8760130,
						"title": "A Stop",
						"gmo": [{"gameBrand":"HOLOHOLO","entity":"POKESTOP","status":"ACTIVE"}]
					},
					{
						"poiId": "empty-gmo",
						"latE6": 50100000,
						"lngE6": 8760000,
						"title": "Business",
						"gmo": []
					},
					{
						"poiId": "active-1",
						"latE6": 50108230,
						"lngE6": 8760796,
						"title": "Active Spot Dup",
						"gmo": [{"gameBrand":"HOLOHOLO","entity":"POWERSPOT","status":"ACTIVE"}]
					}
				]
			}]
		},
		"code": "OK"
	}`)

	pois, err := parsePowerspots(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(pois) != 2 {
		t.Fatalf("got %d pois, want 2: %#v", len(pois), pois)
	}

	byID := map[string]poi.POI{}
	for _, p := range pois {
		byID[p.ID] = p
	}

	active := byID["active-1"]
	if active.Type != poi.TypePowerspot || active.Status != poi.StatusActive || active.Name != "Active Spot" {
		t.Fatalf("active: %#v", active)
	}
	if active.Lat != 50.10823 || active.Lng != 8.760796 {
		t.Fatalf("active coords: %v,%v", active.Lat, active.Lng)
	}

	inactive := byID["inactive-1"]
	if inactive.Status != poi.StatusInactive || inactive.Name != "Inactive Spot" {
		t.Fatalf("inactive: %#v", inactive)
	}
	if inactive.Lat != 50.106375 || inactive.Lng != 8.759598 {
		t.Fatalf("inactive coords: %v,%v", inactive.Lat, inactive.Lng)
	}
}
