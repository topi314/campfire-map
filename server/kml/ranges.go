package kml

import (
	"fmt"
	"math"
	"strings"

	"github.com/topi314/campfire-export/server/poi"
)

const circleSteps = 64

type rangeDef struct {
	key    string
	label  string
	meters float64
	// hex RGB like #e74c3c
	color string
	// nil = every non-route point POI
	types map[poi.Type]bool
}

var exportRanges = []rangeDef{
	{
		key:    "range_wayfarer",
		label:  "Min POI spacing",
		meters: 30,
		color:  "#e74c3c",
		types:  nil,
	},
	{
		key:    "range_lure",
		label:  "Lure",
		meters: 40,
		color:  "#e91e8c",
		types:  map[poi.Type]bool{poi.TypePokeStop: true},
	},
	{
		key:    "range_interaction",
		label:  "Interaction",
		meters: 80,
		color:  "#3da0ff",
		types: map[poi.Type]bool{
			poi.TypeGym:          true,
			poi.TypeSuperMegaGym: true,
			poi.TypePokeStop:     true,
			poi.TypePowerspot:    true,
		},
	},
}

func rangeApplies(def rangeDef, p poi.POI) bool {
	if p.Type == poi.TypeRoute {
		return false
	}
	if def.types == nil {
		return true
	}
	return def.types[p.Type]
}

// hexToKML converts #rrggbb to KML aabbggrr.
func hexToKML(hex string, alpha byte) string {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return fmt.Sprintf("%02x808080", alpha)
	}
	return fmt.Sprintf("%02x%s%s%s", alpha, h[4:6], h[2:4], h[0:2])
}

func destinationPoint(lat, lng, distanceM, bearingDeg float64) (float64, float64) {
	const R = 6371000.0
	br := bearingDeg * math.Pi / 180
	lat1 := lat * math.Pi / 180
	lng1 := lng * math.Pi / 180
	ang := distanceM / R
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(ang) + math.Cos(lat1)*math.Sin(ang)*math.Cos(br))
	lng2 := lng1 + math.Atan2(
		math.Sin(br)*math.Sin(ang)*math.Cos(lat1),
		math.Cos(ang)-math.Sin(lat1)*math.Sin(lat2),
	)
	return lat2 * 180 / math.Pi, lng2 * 180 / math.Pi
}

func circleRing(lat, lng, meters float64) [][2]float64 {
	ring := make([][2]float64, 0, circleSteps+1)
	for i := 0; i < circleSteps; i++ {
		bearing := float64(i) * 360 / float64(circleSteps)
		la, ln := destinationPoint(lat, lng, meters, bearing)
		ring = append(ring, [2]float64{la, ln})
	}
	ring = append(ring, ring[0])
	return ring
}

func circlePlacemark(p poi.POI, def rangeDef) placemark {
	ring := circleRing(p.Lat, p.Lng, def.meters)
	coords := make([]string, 0, len(ring))
	for _, pt := range ring {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
	}
	name := fmt.Sprintf("%s · %s (%gm)", p.Name, def.label, def.meters)
	desc := fmt.Sprintf("%s range · %.6f, %.6f · Type: %s", def.label, p.Lat, p.Lng, def.label)
	return placemark{
		Name:         name,
		Description:  desc,
		StyleURL:     "#" + def.key,
		ExtendedData: typeData(def.label),
		Polygon: &polygon{
			OuterBoundaryIs: outerBoundary{
				LinearRing: linearRing{Coordinates: strings.Join(coords, " ")},
			},
		},
	}
}

