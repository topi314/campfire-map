package kml

// ExportLayer is a named My Maps folder with assigned type keys.
type ExportLayer struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}

// ExportArea is a named polygon ring (lat,lng) for My Maps zone placemarks.
type ExportArea struct {
	Name  string       `json:"name"`
	Color string       `json:"color,omitempty"` // #rrggbb
	Ring  [][2]float64 `json:"ring"`            // lat,lng; closed automatically if needed
}

// ExportOptions controls folder layout and optional map areas.
type ExportOptions struct {
	Layers []ExportLayer
	Areas  []ExportArea
	// Outline is a legacy single ring; used when Areas is empty.
	Outline [][2]float64
}
