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

// ExportRoute is a user-drawn guide path (open polyline) for My Maps line placemarks.
type ExportRoute struct {
	Name  string       `json:"name"`
	Color string       `json:"color,omitempty"` // #rrggbb
	Path  [][2]float64 `json:"path"`            // lat,lng; ≥2
}

// ExportOptions controls folder layout and optional map areas/routes.
type ExportOptions struct {
	Layers []ExportLayer
	Areas  []ExportArea
	Routes []ExportRoute
	// Outline is a legacy single ring; used when Areas is empty.
	Outline [][2]float64
}
