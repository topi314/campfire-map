package kml

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"github.com/topi314/campfire-export/server/poi"
)

//go:embed icons/*.svg
var iconSVGs embed.FS

const iconSize = 64

// Colors match frontend TYPE_META / ROUTE_COLORS.
var typeFill = map[poi.Type]string{
	poi.TypeGym:          "#a4b0bf",
	poi.TypeSuperMegaGym: "", // gradient SVG — leave as-is
	poi.TypePokeStop:     "#7fcafe",
	poi.TypePowerspot:    "#f481c4",
	poi.TypeRoute:        "#01a3ee",
}

func iconPNG(t poi.Type) ([]byte, error) {
	return rasterSVG("icons/"+string(t)+".svg", typeFill[t], false)
}

func routeEndPNG() ([]byte, error) {
	return rasterSVG("icons/route.svg", "#ff4747", true)
}

func rasterSVG(path, fill string, flipX bool) ([]byte, error) {
	raw, err := iconSVGs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("icon %s: %w", path, err)
	}
	svg := string(raw)
	if fill != "" {
		svg = strings.ReplaceAll(svg, "currentColor", fill)
	}
	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	icon.SetTarget(0, 0, float64(iconSize), float64(iconSize))
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	scanner := rasterx.NewScannerGV(iconSize, iconSize, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(iconSize, iconSize, scanner), 1)
	if flipX {
		img = flipRGBAHorizontal(img)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func flipRGBAHorizontal(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	w := b.Dx()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Min.X+w-1-(x-b.Min.X), y, src.At(x, y))
		}
	}
	return dst
}
