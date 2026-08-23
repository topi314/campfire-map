package s2cells

import (
	"strconv"
	"testing"

	"github.com/golang/geo/s2"

	"github.com/topi314/campfire-export/server/poi"
)

func TestCover(t *testing.T) {
	cells, err := Cover(poi.BBox{MinLat: 52.51, MinLng: 13.39, MaxLat: 52.53, MaxLng: 13.41}, 14, 32)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) == 0 {
		t.Fatal("expected cells")
	}
	if cells[0].Level != 14 {
		t.Fatalf("level %d", cells[0].Level)
	}
	if len(cells[0].Ring) < 4 {
		t.Fatalf("expected ring, got %#v", cells[0])
	}
}

func TestCoverIncludesBBoxCorners(t *testing.T) {
	bbox := poi.BBox{MinLat: 50.09, MinLng: 8.75, MaxLat: 50.12, MaxLng: 8.78}
	cells, err := Cover(bbox, 14, 64)
	if err != nil {
		t.Fatal(err)
	}
	pts := [][2]float64{
		{bbox.MinLat, bbox.MinLng},
		{bbox.MinLat, bbox.MaxLng},
		{bbox.MaxLat, bbox.MinLng},
		{bbox.MaxLat, bbox.MaxLng},
		{(bbox.MinLat + bbox.MaxLat) / 2, (bbox.MinLng + bbox.MaxLng) / 2},
	}
	for _, pt := range pts {
		if !pointInCells(t, cells, pt[0], pt[1]) {
			t.Fatalf("point %v not covered by %d cells", pt, len(cells))
		}
	}
}

func TestCoverTrimsTooManyCells(t *testing.T) {
	bbox := poi.BBox{MinLat: 50.0, MinLng: 8.6, MaxLat: 50.2, MaxLng: 8.9}
	cells, err := Cover(bbox, 17, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 8 {
		t.Fatalf("got %d cells, want 8", len(cells))
	}
}

func TestCoverFirstFitPrefersL17(t *testing.T) {
	bbox := poi.BBox{MinLat: 50.100, MinLng: 8.760, MaxLat: 50.104, MaxLng: 8.766}
	cells, level, err := CoverFirstFit(bbox, 128, Level17, Level14)
	if err != nil {
		t.Fatal(err)
	}
	if level != Level17 {
		t.Fatalf("got level %d (%d cells), want 17", level, len(cells))
	}
}

func TestCoverFirstFitFallsBackToL14(t *testing.T) {
	bbox := poi.BBox{MinLat: 50.09, MinLng: 8.75, MaxLat: 50.12, MaxLng: 8.78}
	cells, level, err := CoverFirstFit(bbox, 48, Level17, Level14)
	if err != nil {
		t.Fatal(err)
	}
	if level != Level14 {
		t.Fatalf("got level %d (%d cells), want 14", level, len(cells))
	}
}

func TestOverlayIncludesL14AndL17(t *testing.T) {
	bbox := poi.BBox{MinLat: 50.100, MinLng: 8.760, MaxLat: 50.104, MaxLng: 8.766}
	cells, err := Overlay(bbox)
	if err != nil {
		t.Fatal(err)
	}
	var n14, n17 int
	for _, c := range cells {
		switch c.Level {
		case Level14:
			n14++
		case Level17:
			n17++
		}
	}
	if n14 == 0 || n17 == 0 {
		t.Fatalf("want L14 and L17 overlay, got L14=%d L17=%d", n14, n17)
	}
}

func TestCoverInvalid(t *testing.T) {
	if _, err := Cover(poi.BBox{MinLat: 1, MinLng: 1, MaxLat: 1, MaxLng: 2}, 14, 8); err == nil {
		t.Fatal("expected invalid bbox")
	}
}

func pointInCells(t *testing.T, cells []Cell, lat, lng float64) bool {
	t.Helper()
	p := s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng))
	for _, c := range cells {
		id, err := strconv.ParseUint(c.ID, 10, 64)
		if err != nil {
			t.Fatalf("cell id %s: %v", c.ID, err)
		}
		if s2.CellFromCellID(s2.CellID(id)).ContainsPoint(p) {
			return true
		}
	}
	return false
}
