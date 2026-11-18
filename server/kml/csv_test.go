package kml

import (
	"strings"
	"testing"

	"github.com/topi314/campfire-export/server/poi"
)

func TestBuildCSVIncludesAssignedPois(t *testing.T) {
	pois := []poi.POI{
		{ID: "g1", Type: poi.TypeGym, Name: "Gym One", Lat: 50.1, Lng: 8.7},
		{ID: "c1", Type: poi.TypePokeStop, Name: "=Planned", Lat: 50.2, Lng: 8.8, Source: poi.SourceCampsite},
		{ID: "s1", Type: poi.TypePokeStop, Name: "Unassigned stop", Lat: 1, Lng: 2},
		{ID: "r1", Type: poi.TypeRoute, Name: "Skip route", Lat: 3, Lng: 4, Path: [][2]float64{{3, 4}, {3.1, 4.1}}},
		{ID: "p1", Type: poi.TypePowerspot, Name: "Dark Spot", Lat: 50.3, Lng: 8.9, Status: poi.StatusInactive},
	}
	opts := ExportOptions{Layers: []ExportLayer{
		{Name: "Existing", Types: []string{"gym", "powerspot"}},
		{Name: "Campsite", Types: []string{"campsite_pokestop"}},
	}}
	b, err := BuildCSV(pois, opts)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "\uFEFF") {
		t.Fatal("csv should start with a UTF-8 BOM")
	}
	for _, needle := range []string{
		"name,poi type,latitude,longitude,layer,id,status,source",
		"Gym One,Gym,50.100000,8.700000,Existing,g1,,",
		"'=Planned,Campsite PokéStop,50.200000,8.800000,Campsite,c1,,campsite",
		"Dark Spot,Powerspot · Inactive,50.300000,8.900000,Existing,p1,inactive,",
	} {
		if !strings.Contains(s, needle) {
			t.Errorf("csv missing %q\n%s", needle, s)
		}
	}
	if !strings.Contains(s, "Unassigned stop,PokéStop,1.000000,2.000000,,s1,,") {
		t.Error("POIs without a layer should still be included")
	}
	if strings.Contains(s, "Skip route") || strings.Contains(s, "Route,") {
		t.Errorf("routes should be omitted:\n%s", s)
	}
}

func TestBuildCSVSelectedColumns(t *testing.T) {
	pois := []poi.POI{
		{ID: "g1", Type: poi.TypeGym, Name: "Gym One", Lat: 50.1, Lng: 8.7},
	}
	opts := ExportOptions{
		Layers:     []ExportLayer{{Name: "Existing", Types: []string{"gym"}}},
		CSVColumns: []string{"longitude", "name", "type", "latitude"},
	}
	b, err := BuildCSV(pois, opts)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "longitude,name,poi type,latitude") {
		t.Fatalf("columns should follow the requested order:\n%s", s)
	}
	if !strings.Contains(s, "8.700000,Gym One,Gym,50.100000") {
		t.Fatalf("unexpected row:\n%s", s)
	}
	if strings.Contains(s, "Existing") || strings.Contains(s, "g1") {
		t.Fatalf("unselected columns should be omitted:\n%s", s)
	}
}
