package kml

import (
	"embed"
	"fmt"

	"github.com/topi314/campfire-export/server/poi"
)

// Pre-rasterized icons (Inkscape from frontend/public/images SVGs).
// Colors match frontend TYPE_META / ROUTE_COLORS.
//
//go:embed icons/*.png
var iconPNGs embed.FS

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
