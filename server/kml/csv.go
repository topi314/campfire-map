package kml

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/topi314/campfire-export/server/poi"
)

// BuildCSV writes one row per POI. Routes, areas, drawn routes, and range circles are omitted.
func BuildCSV(pois []poi.POI, opts ExportOptions) ([]byte, error) {
	layers := opts.Layers
	if len(layers) == 0 {
		layers = defaultLayers()
	}
	typeToLayer := map[string]int{}
	for i, layer := range layers {
		for _, t := range layer.Types {
			if _, exists := typeToLayer[t]; !exists {
				typeToLayer[t] = i
			}
		}
	}

	columns, err := resolveCSVColumns(opts.CSVColumns)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	header := make([]string, len(columns))
	for i, col := range columns {
		header[i] = col.header
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, p := range pois {
		if p.Type == poi.TypeRoute {
			continue
		}
		layerName := ""
		if idx, ok := typeToLayer[p.LayerTypeKey()]; ok {
			layerName = strings.TrimSpace(layers[idx].Name)
			if layerName == "" {
				layerName = fmt.Sprintf("Layer %d", idx+1)
			}
		}
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = csvColumnValue(col.key, p, layerName)
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type csvColumn struct {
	key    string
	header string
}

var csvColumnCatalog = []csvColumn{
	{"name", "name"},
	{"type", "poi type"},
	{"latitude", "latitude"},
	{"longitude", "longitude"},
	{"layer", "layer"},
	{"id", "id"},
	{"status", "status"},
	{"source", "source"},
}

func resolveCSVColumns(requested []string) ([]csvColumn, error) {
	byKey := map[string]csvColumn{}
	for _, col := range csvColumnCatalog {
		byKey[col.key] = col
	}
	if len(requested) == 0 {
		return append([]csvColumn(nil), csvColumnCatalog...), nil
	}
	var out []csvColumn
	seen := map[string]bool{}
	for _, key := range requested {
		key = strings.ToLower(strings.TrimSpace(key))
		col, ok := byKey[key]
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, col)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no csv columns selected")
	}
	return out, nil
}

func csvColumnValue(key string, p poi.POI, layerName string) string {
	switch key {
	case "name":
		return csvText(p.Name)
	case "type":
		return csvText(typeColumn(p))
	case "latitude":
		return fmt.Sprintf("%.6f", p.Lat)
	case "longitude":
		return fmt.Sprintf("%.6f", p.Lng)
	case "layer":
		return csvText(layerName)
	case "id":
		return csvText(p.ID)
	case "status":
		return csvText(p.Status)
	case "source":
		return csvText(p.Source)
	default:
		return ""
	}
}

// csvText keeps spreadsheet apps from treating a text cell as a formula.
func csvText(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	default:
		return s
	}
}
