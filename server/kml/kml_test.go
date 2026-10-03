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
		{ID: "c1", Type: poi.TypePokeStop, Name: "Planned Stop", Lat: 52.52, Lng: 13.42, Source: poi.SourceCampsite},
	}, ExportOptions{
		Layers: []ExportLayer{
			{Name: "Existing", Types: []string{"gym", "super_mega_gym", "pokestop", "powerspot", "route"}},
			{Name: "Campsite", Types: []string{"campsite_gym", "campsite_pokestop", "campsite_powerspot", "outline"}},
		},
		Outline: [][2]float64{{52.5, 13.4}, {52.51, 13.4}, {52.51, 13.41}},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var kmlDoc string
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
			kmlDoc = string(b)
		}
	}
	if !files["doc.kml"] || !files["files/gym.png"] || !files["files/route.png"] || !files["files/route-end.png"] || !files["files/super_mega_gym.png"] || !files["files/campsite_pokestop.png"] {
		t.Fatalf("missing kmz entries: %v", files)
	}
	for _, needle := range []string{
		"Existing", "Campsite",
		"Gym One", "Mega Gate", "Loop", "Planned Stop", "Draw area",
		"<LineString>", "<Point>", "<Polygon>", "#super_mega_gym", `<Style id="gym"`, "files/gym.png",
		`#campsite_pokestop`, `files/campsite_pokestop.png`, `<Style id="campsite_pokestop"`,
		"route-line", "#outline",
	} {
		if !strings.Contains(kmlDoc, needle) {
			t.Errorf("kml missing %q", needle)
		}
	}
	// Routes must not create separate path/end folders.
	for _, bad := range []string{"Route paths", "Route ends"} {
		if strings.Contains(kmlDoc, bad) {
			t.Errorf("unexpected separate route folder %q", bad)
		}
	}
	// Shared styles should reference embedded icons, not remote URLs.
	if strings.Contains(kmlDoc, "https://example.com") {
		t.Error("kml should not embed remote POI photo URLs in styles")
	}
	// Existing folder should contain route start + path + end (mixed styleUrls OK).
	if !folderContains(kmlDoc, "Existing", "#route") || !folderContains(kmlDoc, "Existing", "#route-line") || !folderContains(kmlDoc, "Existing", "#route-end") {
		t.Error("route start/path/end should all be in Existing folder")
	}
	if !folderContains(kmlDoc, "Campsite", "Planned Stop") || !folderContains(kmlDoc, "Campsite", "Draw area") {
		t.Error("campsite POI and outline should be in Campsite folder")
	}
}

func TestBuildKML(t *testing.T) {
	data, err := BuildKML("Test map", []poi.POI{
		{ID: "g1", Type: poi.TypeGym, Name: "Gym One", Lat: 52.5, Lng: 13.4},
	}, ExportOptions{})
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
	if !strings.Contains(s, "Gym One") || !strings.Contains(s, `<Style id="gym"`) {
		t.Fatalf("unexpected kml:\n%s", s[:min(400, len(s))])
	}
}

func folderContains(kmlDoc, folderName, needle string) bool {
	parts := strings.Split(kmlDoc, "<Folder>")
	for _, part := range parts[1:] {
		end := strings.Index(part, "</Folder>")
		if end < 0 {
			continue
		}
		body := part[:end]
		if strings.Contains(body, "<name>"+folderName+"</name>") && strings.Contains(body, needle) {
			return true
		}
	}
	return false
}

func TestFolderSplit(t *testing.T) {
	pois := make([]poi.POI, featuresPerFolder+3)
	for i := range pois {
		pois[i] = poi.POI{ID: fmt.Sprintf("s-%d", i), Type: poi.TypePokeStop, Name: "S", Lat: 1, Lng: 1}
	}
	data, err := BuildKMZ("split", pois, ExportOptions{
		Layers: []ExportLayer{{Name: "PokéStops", Types: []string{"pokestop"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var kmlDoc string
	for _, f := range zr.File {
		if f.Name == "doc.kml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			_ = rc.Close()
			kmlDoc = string(b)
		}
	}
	if !strings.Contains(kmlDoc, "PokéStops 2") {
		t.Fatalf("expected overflow folder, got:\n%s", kmlDoc[:min(800, len(kmlDoc))])
	}
}

func TestSharedStyleByType(t *testing.T) {
	data, err := BuildKML("styles", []poi.POI{
		{ID: "a", Type: poi.TypeGym, Name: "A", Lat: 1, Lng: 1},
		{ID: "b", Type: poi.TypeGym, Name: "B", Lat: 2, Lng: 2},
		{ID: "c", Type: poi.TypePokeStop, Name: "C", Lat: 3, Lng: 3},
	}, ExportOptions{
		Layers: []ExportLayer{{Name: "Mixed", Types: []string{"gym", "pokestop"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Count(s, "<styleUrl>#gym</styleUrl>") < 2 {
		t.Fatal("expected shared #gym styleUrl for both gyms")
	}
	if !strings.Contains(s, "<styleUrl>#pokestop</styleUrl>") {
		t.Fatal("expected #pokestop styleUrl")
	}
}
