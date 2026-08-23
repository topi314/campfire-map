package s2cells

import (
	"fmt"
	"sort"

	"github.com/golang/geo/s2"

	"github.com/topi314/campfire-export/server/poi"
)

const (
	Level14 = 14 // Pokémon GO gym / PokéStop parent cells
	Level15 = 15 // Campfire mapObjectsByS2Cells (full POI details)
	Level17 = 17 // Pokémon GO POI cells (64 per L14 cell)

	MaxOverlay14 = 256
	MaxOverlay17 = 1024
)

type Cell struct {
	ID    string       `json:"id"`
	Level int          `json:"level"`
	Ring  [][2]float64 `json:"ring"`
}

func Cover(bbox poi.BBox, level int, maxCells int) ([]Cell, error) {
	cells, _, err := CoverFirstFit(bbox, maxCells, level)
	return cells, err
}

// CoverFirstFit covers bbox with the first level that fits in maxCells.
// If every level exceeds maxCells, it keeps the cells nearest the bbox center.
func CoverFirstFit(bbox poi.BBox, maxCells int, levels ...int) ([]Cell, int, error) {
	if err := bbox.Valid(); err != nil {
		return nil, 0, err
	}
	if maxCells < 1 {
		maxCells = 32
	}
	if len(levels) == 0 {
		levels = []int{Level15}
	}

	rect := latLngRect(bbox)
	clat, clng := bbox.Center()
	center := s2.LatLngFromDegrees(clat, clng)

	var bestCovering s2.CellUnion
	var bestLevel int
	for _, level := range levels {
		if level <= 0 || level > 30 {
			continue
		}
		covering := coveringAt(rect, level)
		if len(covering) == 0 {
			continue
		}
		bestCovering = covering
		bestLevel = level
		if len(covering) <= maxCells {
			return toCells(covering, level), level, nil
		}
	}
	if len(bestCovering) == 0 {
		return nil, 0, fmt.Errorf("no S2 cells for bbox")
	}
	return toCells(trimNearest(bestCovering, maxCells, center), bestLevel), bestLevel, nil
}

func trimNearest(covering s2.CellUnion, maxCells int, center s2.LatLng) s2.CellUnion {
	if len(covering) <= maxCells {
		return covering
	}
	type ranked struct {
		id s2.CellID
		d  float64
	}
	centerPt := s2.PointFromLatLng(center)
	items := make([]ranked, len(covering))
	for i, id := range covering {
		cell := s2.CellFromCellID(id)
		items[i] = ranked{id: id, d: float64(centerPt.Distance(cell.Center()))}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].d == items[j].d {
			return uint64(items[i].id) < uint64(items[j].id)
		}
		return items[i].d < items[j].d
	})
	out := make(s2.CellUnion, maxCells)
	for i := 0; i < maxCells; i++ {
		out[i] = items[i].id
	}
	return out
}

// Overlay returns L14 cells plus L17 cells when the viewport is small enough
// to draw them.
func Overlay(bbox poi.BBox) ([]Cell, error) {
	if err := bbox.Valid(); err != nil {
		return nil, err
	}
	rect := latLngRect(bbox)
	var out []Cell

	l14 := coveringAt(rect, Level14)
	if len(l14) > 0 && len(l14) <= MaxOverlay14 {
		out = append(out, toCells(l14, Level14)...)
	}
	l17 := coveringAt(rect, Level17)
	if len(l17) > 0 && len(l17) <= MaxOverlay17 {
		out = append(out, toCells(l17, Level17)...)
	}
	return out, nil
}

func latLngRect(bbox poi.BBox) s2.Rect {
	lo := s2.LatLngFromDegrees(bbox.MinLat, bbox.MinLng)
	hi := s2.LatLngFromDegrees(bbox.MaxLat, bbox.MaxLng)
	rect := s2.RectFromLatLng(lo)
	return rect.AddPoint(hi)
}

func coveringAt(rect s2.Rect, level int) s2.CellUnion {
	rc := &s2.RegionCoverer{
		MinLevel: level,
		MaxLevel: level,
		MaxCells: 1 << 20,
	}
	return rc.Covering(rect)
}

func toCells(covering s2.CellUnion, level int) []Cell {
	cells := make([]Cell, 0, len(covering))
	for _, cid := range covering {
		cell := s2.CellFromCellID(cid)
		ring := make([][2]float64, 0, 5)
		for i := 0; i < 4; i++ {
			ll := s2.LatLngFromPoint(cell.Vertex(i))
			ring = append(ring, [2]float64{ll.Lat.Degrees(), ll.Lng.Degrees()})
		}
		ring = append(ring, ring[0])
		cells = append(cells, Cell{
			ID:    fmt.Sprintf("%d", uint64(cid)),
			Level: level,
			Ring:  ring,
		})
	}
	return cells
}
