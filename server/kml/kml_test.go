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
			{Name: "Ranges", Types: []string{"range_wayfarer"}},
			{Name: "Campsite", Types: []string{"campsite_gym", "campsite_pokestop", "campsite_powerspot", "outline"}},
		},
		Areas: []ExportArea{
			{Name: "Play Area", Color: "#5b8cff", Ring: [][2]float64{{52.5, 13.4}, {52.51, 13.4}, {52.51, 13.41}}},
			{Name: "Staging", Color: "#e85d4c", Ring: [][2]float64{{52.52, 13.4}, {52.53, 13.4}, {52.53, 13.41}}},
		},
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
		"Existing", "Campsite", "Ranges",
		"Gym One", "Mega Gate", "Loop", "Planned Stop", "Play Area", "Staging",
		"Min POI spacing",
		"<LineString>", "<Point>", "<Polygon>", "#super_mega_gym", "files/gym.png",
		`#campsite_pokestop`, `files/campsite_pokestop.png`,
		"route-line", `#poly-5B8CFF-3000-64`, `#poly-E85D4C-3000-64`, `#range_wayfarer`,
		`<Style id="gym-normal"`, `<StyleMap id="gym"`,
		`<Style id="campsite_pokestop-normal"`, `<StyleMap id="campsite_pokestop"`,
		`<Style id="range_wayfarer-normal"`, `<StyleMap id="range_wayfarer"`,
		`<StyleMap id="poly-E85D4C-3000-64"`,
		`<width>4.5</width>`, // My Maps poly highlight width
		`<Data name="Type">`, "<value>Gym</value>", "<value>Campsite PokéStop</value>",
		"<value>Min POI spacing</value>", "<value>Play Area</value>", "<value>Staging</value>",
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
	// Open route "Loop" should include start + path + end.
	if !folderContains(kmlDoc, "Existing", "#route") || !folderContains(kmlDoc, "Existing", "#route-line") || !folderContains(kmlDoc, "Existing", "#route-end") {
		t.Error("open route start/path/end should all be in Existing folder")
	}
	if !folderContains(kmlDoc, "Campsite", "Planned Stop") || !folderContains(kmlDoc, "Campsite", "Play Area") || !folderContains(kmlDoc, "Campsite", "Staging") {
		t.Error("campsite POI and all areas should be in one Campsite folder")
	}
	if !folderContains(kmlDoc, "Campsite", "#poly-5B8CFF-3000-64") || !folderContains(kmlDoc, "Campsite", "#poly-E85D4C-3000-64") {
		t.Error("areas should use per-color My Maps poly-* styleUrls in the shared layer")
	}
	if !folderContains(kmlDoc, "Ranges", "#range_wayfarer") {
		t.Error("range circles should be in Ranges folder")
	}
}

func TestLegacyOutlineExport(t *testing.T) {
	data, err := BuildKML("legacy", nil, ExportOptions{
		Layers:  []ExportLayer{{Name: "Areas", Types: []string{"outline"}}},
		Outline: [][2]float64{{1, 2}, {1.1, 2}, {1.1, 2.1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Play Area") || !strings.Contains(s, "<Polygon>") || !strings.Contains(s, "Areas") {
		t.Fatalf("expected Areas folder + Play Area placemark, got:\n%s", s[:min(400, len(s))])
	}
}

func TestDrawnRouteExport(t *testing.T) {
	data, err := BuildKML("guides", nil, ExportOptions{
		Layers: []ExportLayer{{Name: "Campsite Routes", Types: []string{"drawn_route"}}},
		Routes: []ExportRoute{
			{
				Name:  "Trail A",
				Color: "#e85d4c",
				Path:  [][2]float64{{52.5, 13.4}, {52.51, 13.41}, {52.52, 13.42}},
			},
			{
				Name:  "Raid Route",
				Color: "#3dbb7a",
				Path:  [][2]float64{{52.6, 13.4}, {52.61, 13.41}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	wantLine := hexToKML("#e85d4c", 0xff)
	wantGreen := hexToKML("#3dbb7a", 0xff) // matches working Untitled map(1).kmz
	for _, needle := range []string{
		"Campsite Routes",
		"Trail A",
		"Raid Route",
		"<LineString>",
		"#route",
		"#route-end",
		`#line-E85D4C-4000`,
		`#line-3DBB7A-4000`,
		`<StyleMap id="line-E85D4C-4000"`,
		`<StyleMap id="line-3DBB7A-4000"`,
		wantLine,
		wantGreen,
		`<width>6</width>`, // My Maps line highlight = 4 × 1.5
		"<value>Trail A</value>",
		`<Data name="description">`,
	} {
		if !strings.Contains(s, needle) {
			t.Errorf("kml missing %q", needle)
		}
	}
	if strings.Contains(s, "drawn_route") {
		t.Error("drawn routes should use My Maps line-RRGGBB-4000 ids, not drawn_route*")
	}
	if !folderContains(s, "Campsite Routes", "<LineString>") ||
		!folderContains(s, "Campsite Routes", "#route") ||
		!folderContains(s, "Campsite Routes", "#route-end") ||
		!folderContains(s, "Campsite Routes", "#line-E85D4C-4000") ||
		!folderContains(s, "Campsite Routes", "#line-3DBB7A-4000") {
		t.Error("open drawn route start/path/end should be in Campsite Routes folder with per-color line styles")
	}
}

func TestDefaultOutlineColorABGR(t *testing.T) {
	// #5b8cff → aabbggrr ffff8c5b / 40ff8c5b (not HTML-order 405b8cff).
	wantLine := hexToKML("#5b8cff", 0xff)
	wantFill := hexToKML("#5b8cff", 0x40)
	if wantLine != "ffff8c5b" || wantFill != "40ff8c5b" {
		t.Fatalf("hexToKML unexpected: line=%s fill=%s", wantLine, wantFill)
	}
	data, err := BuildKML("colors", nil, ExportOptions{
		Layers: []ExportLayer{{Name: "Areas", Types: []string{"outline"}}},
		Areas: []ExportArea{
			{Name: "Default", Color: "#5b8cff", Ring: [][2]float64{{1, 2}, {1.1, 2}, {1.1, 2.1}}},
			{Name: "Custom", Color: "#e85d4c", Ring: [][2]float64{{3, 4}, {3.1, 4}, {3.1, 4.1}}},
			{Name: "Indigo", Color: "#1a237e", Ring: [][2]float64{{5, 6}, {5.1, 6}, {5.1, 6.1}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, wantLine) || !strings.Contains(s, wantFill) {
		t.Errorf("default outline missing correct aabbggrr colors %s / %s", wantLine, wantFill)
	}
	if strings.Contains(s, "405b8cff") || strings.Contains(s, "ff8cff5b") {
		t.Error("default outline still uses HTML-order KML color bytes")
	}
	for _, needle := range []string{
		`#poly-5B8CFF-3000-64`,
		`#poly-E85D4C-3000-64`,
		`#poly-1A237E-3000-64`,
		hexToKML("#e85d4c", 0xff),
		hexToKML("#1a237e", 0x40),
		`<width>4.5</width>`,
	} {
		if !strings.Contains(s, needle) {
			t.Errorf("kml missing %q", needle)
		}
	}
}

func TestClosedRouteOmitsEndMarker(t *testing.T) {
	loop := [][2]float64{{52.5, 13.4}, {52.51, 13.4}, {52.51, 13.41}, {52.5, 13.4}}
	data, err := BuildKML("loop", []poi.POI{
		{ID: "r1", Type: poi.TypeRoute, Name: "Circuit", Lat: 52.5, Lng: 13.4, Path: loop},
	}, ExportOptions{
		Layers: []ExportLayer{
			{Name: "Routes", Types: []string{"route"}},
			{Name: "Campsite Routes", Types: []string{"drawn_route"}},
		},
		Routes: []ExportRoute{{Name: "Drawn Loop", Color: "#5b8cff", Path: loop}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Circuit") || !strings.Contains(s, "Drawn Loop") {
		t.Fatal("expected both closed routes in export")
	}
	if !strings.Contains(s, "<styleUrl>#route</styleUrl>") || !strings.Contains(s, "#route-line") {
		t.Fatal("expected start + path for closed route")
	}
	// Shared #route-end Style may exist; placemarks must not reference it.
	if strings.Contains(s, "<styleUrl>#route-end</styleUrl>") || strings.Contains(s, "(end)") {
		t.Error("closed loop should omit end marker placemark")
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
	if !strings.Contains(s, "Gym One") || !strings.Contains(s, `<StyleMap id="gym"`) || !strings.Contains(s, `<Style id="gym-normal"`) {
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
