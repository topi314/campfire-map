package kml

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/topi314/campfire-export/server/poi"
)

// Pre-rasterized icons (Inkscape from frontend/public/images SVGs).
// Colors match frontend TYPE_META / ROUTE_COLORS.
//
//go:embed icons/*.png
var iconPNGs embed.FS

// Amber tint matching frontend CAMPSITE_MARKER (#f0a202).
var campsiteAmber = color.NRGBA{R: 0xf0, G: 0xa2, B: 0x02, A: 0xff}

// Campsite style/file keys → base POI type glyph.
var campsiteIconBases = map[string]poi.Type{
	"campsite_gym":       poi.TypeGym,
	"campsite_pokestop":  poi.TypePokeStop,
	"campsite_powerspot": poi.TypePowerspot,
}

func iconPNG(t poi.Type) ([]byte, error) {
	b, err := iconPNGs.ReadFile("icons/" + string(t) + ".png")
	if err != nil {
		return nil, fmt.Errorf("icon %s: %w", t, err)
	}
	return b, nil
}

func routeEndPNG() ([]byte, error) {
	b, err := iconPNGs.ReadFile("icons/route-end.png")
	if err != nil {
		return nil, fmt.Errorf("icon route-end: %w", err)
	}
	return b, nil
}

func campsiteIconPNG(key string) ([]byte, error) {
	base, ok := campsiteIconBases[key]
	if !ok {
		return nil, fmt.Errorf("unknown campsite icon %q", key)
	}
	src, err := iconPNG(base)
	if err != nil {
		return nil, err
	}
	tinted, err := tintIconPNG(src, campsiteAmber)
	if err != nil {
		return nil, err
	}
	// Small corner badge so My Maps thumbnails stay distinct from live POI icons.
	return badgeIconPNG(tinted, campsiteAmber)
}

// tintIconPNG recolors opaque glyph pixels to tint while preserving alpha edges.
func tintIconPNG(src []byte, tint color.NRGBA) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{R: tint.R, G: tint.G, B: tint.B, A: uint8(a >> 8)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// badgeIconPNG paints a filled circle in the bottom-right so campsite icons read clearly at small sizes.
func badgeIconPNG(src []byte, tint color.NRGBA) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)

	w, h := b.Dx(), b.Dy()
	r := w / 5
	if r < 6 {
		r = 6
	}
	cx := b.Min.X + w - r - 2
	cy := b.Min.Y + h - r - 2
	r2 := r * r
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	for y := cy - r; y <= cy + r; y++ {
		for x := cx - r; x <= cx + r; x++ {
			dx, dy := x-cx, y-cy
			d2 := dx*dx + dy*dy
			if d2 > r2 {
				continue
			}
			// Outer ring white, inner filled tint.
			inner := (r - 2) * (r - 2)
			if d2 > inner {
				out.SetNRGBA(x, y, white)
			} else {
				out.SetNRGBA(x, y, tint)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
