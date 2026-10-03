package kml

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/topi314/campfire-export/server/poi"
)

func TestCampsiteIconTintedAmber(t *testing.T) {
	gym, err := iconPNG(poi.TypeGym)
	if err != nil {
		t.Fatal(err)
	}
	camp, err := campsiteIconPNG("campsite_gym")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(gym, camp) {
		t.Fatal("campsite gym icon should differ from existing gym icon")
	}
	img, err := png.Decode(bytes.NewReader(camp))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	var sumR, sumG, sumB, n int64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a < 128<<8 {
				continue
			}
			sumR += int64(r >> 8)
			sumG += int64(g >> 8)
			sumB += int64(bl >> 8)
			n++
		}
	}
	if n == 0 {
		t.Fatal("no opaque pixels in campsite icon")
	}
	avgR, avgG, avgB := sumR/n, sumG/n, sumB/n
	// Amber #f0a202 — red dominant, green mid, blue low.
	if avgR < 200 || avgG < 120 || avgB > 80 {
		t.Fatalf("campsite icon not amber-ish: avg RGB=%d,%d,%d", avgR, avgG, avgB)
	}
}

func TestCampsitePlacemarkUsesCampsiteStyle(t *testing.T) {
	data, err := BuildKML("t", []poi.POI{
		{ID: "c", Type: poi.TypeGym, Name: "Planned Gym", Lat: 1, Lng: 2, Source: poi.SourceCampsite},
	}, ExportOptions{
		Layers: []ExportLayer{{Name: "Campsite", Types: []string{"campsite_gym"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !bytes.Contains(data, []byte(`#campsite_gym`)) {
		t.Fatalf("expected campsite_gym styleUrl, kml:\n%s", s)
	}
	if bytes.Contains(data, []byte(`<styleUrl>#gym</styleUrl>`)) {
		t.Fatal("planned campsite gym should not use #gym style")
	}
}
