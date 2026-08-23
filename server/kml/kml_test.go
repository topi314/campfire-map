package kml

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/topi314/campfire-export/server/poi"
)

func TestBuildKMZ(t *testing.T) {
	data, err := BuildKMZ("Test map", []poi.POI{
		{ID: "g1", Type: poi.TypeGym, Name: "Gym One", Lat: 52.5, Lng: 13.4, Icon: "https://example.com/g.png"},
		{ID: "sm1", Type: poi.TypeSuperMegaGym, Name: "Mega Gate", Lat: 52.51, Lng: 13.41},
		{ID: "r1", Type: poi.TypeRoute, Name: "Loop", Lat: 52.5, Lng: 13.4, Path: [][2]float64{{52.5, 13.4}, {52.51, 13.41}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var kml string
	files := map[string]bool{}
	for _, f := range zr.File {
		files[f.Name] = true
		if f.Name == "doc.kml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			kml = string(b)
		}
	}
	if !files["doc.kml"] || !files["files/gym.png"] || !files["files/route.png"] || !files["files/route-end.png"] || !files["files/super_mega_gym.png"] {
		t.Fatalf("missing kmz entries: %v", files)
	}
	for _, needle := range []string{
		"Gyms", "Super Mega Gyms", "Routes", "Gym One", "Mega Gate", "Loop",
		"<LineString>", "<Point>", "#super_mega_gym", "<StyleMap", "files/gym.png", "route-line",
	} {
		if !strings.Contains(kml, needle) {
			t.Errorf("kml missing %q", needle)
		}
	}
	// Shared styles should reference embedded icons, not remote URLs.
	if strings.Contains(kml, "https://example.com") {
		t.Error("kml should not embed remote POI photo URLs in styles")
	}
}

func TestBuildKML(t *testing.T) {
	data, err := BuildKML("Test map", []poi.POI{
		{ID: "g1", Type: poi.TypeGym, Name: "Gym One", Lat: 52.5, Lng: 13.4},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "<?xml") {
		t.Fatal("expected xml declaration")
	}
	if !strings.Contains(s, "data:image/png;base64,") {
		t.Fatal("expected inlined icon data URIs")
	}
	if !strings.Contains(s, "Gym One") || !strings.Contains(s, "<StyleMap") {
		t.Fatalf("unexpected kml:\n%s", s[:min(400, len(s))])
	}
}

func TestFolderSplit(t *testing.T) {
	pois := make([]poi.POI, featuresPerFolder+3)
	for i := range pois {
		pois[i] = poi.POI{ID: fmt.Sprintf("s-%d", i), Type: poi.TypePokeStop, Name: "S", Lat: 1, Lng: 1}
	}
	data, err := BuildKMZ("split", pois)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var kml string
	for _, f := range zr.File {
		if f.Name == "doc.kml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			kml = string(b)
		}
	}
	if !strings.Contains(kml, "PokéStops 2") {
		t.Fatalf("expected overflow folder, got:\n%s", kml[:min(800, len(kml))])
	}
}
