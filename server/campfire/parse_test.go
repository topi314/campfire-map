package campfire

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/topi314/campfire-export/server/poi"
)

func TestParseS2CellsQuery(t *testing.T) {
	raw := json.RawMessage(`{
		"realityChannelMapObjectsByS2Cells": {
			"mapObjectsByS2CellsAndTypes": [{
				"s2CellId": "5169303185835163648",
				"mapObjectsByType": [{
					"type": "PGO_GYM",
					"mapObjects": [{
						"mapObjectType": "PGO_GYM",
						"pgoGym": {
							"name": "Park Gym",
							"imageUrl": "https://example.com/gym.png",
							"location": {"latitude": 52.52, "longitude": 13.40},
							"isMegaEnhancedEligible": false
						}
					}, {
						"mapObjectType": "PGO_GYM",
						"pgoGym": {
							"name": "Mega Gym",
							"location": {"latitude": 52.521, "longitude": 13.401},
							"isMegaEnhancedEligible": true
						}
					}]
				}, {
					"type": "PGO_POKESTOP",
					"mapObjects": [{
						"mapObjectType": "PGO_POKESTOP",
						"pgoPokestop": {
							"name": "Museum Stop",
							"imageUrl": "https://example.com/stop.png",
							"location": {"latitude": 52.522, "longitude": 13.402}
						}
					}]
				}, {
					"type": "PGO_POWERSPOT",
					"mapObjects": [{
						"mapObjectType": "PGO_POWERSPOT",
						"pgoPowerspot": {
							"name": "Plaza Spot",
							"location": {"latitude": 52.523, "longitude": 13.403}
						}
					}]
				}, {
					"type": "PGO_ROUTE",
					"mapObjects": [{
						"mapObjectType": "PGO_ROUTE",
						"pgoRoute": {
							"id": "route-1",
							"name": "Park Loop",
							"startPoi": {
								"imageUrl": "https://example.com/route.png",
								"location": {"latitude": 52.524, "longitude": 13.404}
							},
							"endPoi": {"location": {"latitude": 52.525, "longitude": 13.405}},
							"locationList": [
								{"latitude": 52.524, "longitude": 13.404},
								{"latitude": 52.5245, "longitude": 13.4045},
								{"latitude": 52.525, "longitude": 13.405}
							]
						}
					}]
				}]
			}]
		}
	}`)
	pois, err := parseMapObjects(raw)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]poi.POI{}
	for _, p := range pois {
		byID[p.ID] = p
	}
	wantIDs := []string{
		"gym:52.520000,13.400000",
		"gym:52.521000,13.401000",
		"stop:52.522000,13.402000",
		"powerspot:52.523000,13.403000",
		"route-1",
	}
	if len(pois) != len(wantIDs) {
		t.Fatalf("got %d pois %#v, want %d", len(pois), byID, len(wantIDs))
	}
	gym := byID["gym:52.520000,13.400000"]
	if gym.Type != poi.TypeGym || gym.Name != "Park Gym" || gym.Icon != "https://example.com/gym.png" {
		t.Fatalf("gym fields: %#v", gym)
	}
	mega := byID["gym:52.521000,13.401000"]
	if mega.Type != poi.TypeGym || !mega.SuperMegaEligible {
		t.Fatalf("mega gym: %#v", mega)
	}
	stop := byID["stop:52.522000,13.402000"]
	if stop.Type != poi.TypePokeStop || stop.Name != "Museum Stop" {
		t.Fatalf("stop: %#v", stop)
	}
	spot := byID["powerspot:52.523000,13.403000"]
	if spot.Type != poi.TypePowerspot || spot.Name != "Plaza Spot" {
		t.Fatalf("powerspot: %#v", spot)
	}
	route := byID["route-1"]
	if route.Type != poi.TypeRoute || route.Name != "Park Loop" || len(route.Path) != 3 || route.Icon != "https://example.com/route.png" {
		t.Fatalf("route: %#v", route)
	}
}

func TestParseMegaFlagShapes(t *testing.T) {
	cases := []struct {
		name string
		flag string
	}{
		{"bool", `"isMegaEnhancedEligible": true`},
		{"enum", `"isMegaEnhancedEligible": "ELIGIBLE"`},
		{"number", `"isMegaEnhancedEligible": 1`},
		{"nested", `"isMegaEnhancedEligible": {"eligible": true}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(fmt.Sprintf(`{
				"realityChannelMapObjectsByS2Cells": {
					"mapObjectsByS2CellsAndTypes": [{
						"mapObjectsByType": [{
							"type": "PGO_GYM",
							"mapObjects": [{
								"mapObjectType": "PGO_GYM",
								"pgoGym": {
									"name": "Mega",
									"location": {"latitude": 1, "longitude": 2},
									%s
								}
							}]
						}]
					}]
				}
			}`, tc.flag))
			pois, err := parseMapObjects(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(pois) != 1 || pois[0].Type != poi.TypeGym || !pois[0].SuperMegaEligible {
				t.Fatalf("got %#v", pois)
			}
		})
	}
}

func TestMergePOI(t *testing.T) {
	cur := poi.POI{ID: "gym-1", Type: "gym", Name: "Gym", Lat: 1, Lng: 2}
	upd := poi.POI{ID: "gym-1", Type: "gym", Name: "Real Gym", Icon: "https://x", Lat: 1, Lng: 2}
	merged := mergePOI(cur, upd)
	if merged.Name != "Real Gym" || merged.Icon != "https://x" {
		t.Fatalf("got %#v", merged)
	}
}

func TestMergeUpgradesGymToSuperMega(t *testing.T) {
	cur := poi.POI{ID: "gym-1", Type: poi.TypeGym, Name: "Gym", Lat: 1, Lng: 2}
	upd := poi.POI{ID: "gym-1", Type: poi.TypeGym, SuperMegaEligible: true, Name: "Gym", Lat: 1, Lng: 2}
	merged := mergePOI(cur, upd)
	if merged.Type != poi.TypeGym || !merged.SuperMegaEligible {
		t.Fatalf("got %#v", merged)
	}
	legacy := mergePOI(cur, poi.POI{ID: "gym-1", Type: poi.TypeSuperMegaGym, Name: "Gym", Lat: 1, Lng: 2})
	if legacy.Type != poi.TypeGym || !legacy.SuperMegaEligible {
		t.Fatalf("legacy %#v", legacy)
	}
}
