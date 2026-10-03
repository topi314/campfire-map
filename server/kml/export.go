package kml

// ExportLayer is a named My Maps folder with assigned type keys.
type ExportLayer struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}

// ExportOptions controls folder layout and optional campsite outline.
type ExportOptions struct {
	Layers  []ExportLayer
	Outline [][2]float64 // lat,lng rings; closed automatically if needed
}
