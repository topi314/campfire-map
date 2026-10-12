package kml

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/topi314/campfire-export/server/poi"
)

const featuresPerFolder = 2000

// KML colors are aabbggrr. Matches frontend TYPE_META / ROUTE_COLORS.
const (
	routeLineKML = "ffffa03d" // #3da0ff
	iconScale    = "1"
)

type kmlRoot struct {
	XMLName  xml.Name `xml:"kml"`
	Xmlns    string   `xml:"xmlns,attr"`
	Document document `xml:"Document"`
}

type document struct {
	Name      string     `xml:"name"`
	Styles    []style    `xml:"Style"`
	StyleMaps []styleMap `xml:"StyleMap"`
	Folder    []folder   `xml:"Folder"`
}

type style struct {
	ID          string       `xml:"id,attr"`
	IconStyle   *iconStyle   `xml:"IconStyle,omitempty"`
	LabelStyle  *labelStyle  `xml:"LabelStyle,omitempty"`
	LineStyle   *lineStyle   `xml:"LineStyle,omitempty"`
	PolyStyle   *polyStyle   `xml:"PolyStyle,omitempty"`
}

type iconStyle struct {
	Scale   string  `xml:"scale"`
	Icon    icon    `xml:"Icon"`
	HotSpot hotSpot `xml:"hotSpot"`
}

type labelStyle struct {
	Scale string `xml:"scale"`
}

type icon struct {
	Href string `xml:"href"`
}

type hotSpot struct {
	X      string `xml:"x,attr"`
	Y      string `xml:"y,attr"`
	XUnits string `xml:"xunits,attr"`
	YUnits string `xml:"yunits,attr"`
}

type lineStyle struct {
	Color string `xml:"color"`
	Width int    `xml:"width"`
}

type polyStyle struct {
	Color   string `xml:"color"`
	Fill    int    `xml:"fill"`
	Outline int    `xml:"outline"`
}

type styleMap struct {
	ID   string      `xml:"id,attr"`
	Pair []stylePair `xml:"Pair"`
}

type stylePair struct {
	Key      string `xml:"key"`
	StyleURL string `xml:"styleUrl"`
}

type folder struct {
	XMLName    xml.Name    `xml:"Folder"`
	Name       string      `xml:"name"`
	Placemarks []placemark `xml:"Placemark"`
}

type placemark struct {
	Name         string        `xml:"name"`
	Description  string        `xml:"description"`
	StyleURL     string        `xml:"styleUrl"`
	ExtendedData *extendedData `xml:"ExtendedData,omitempty"`
	Point        *point        `xml:"Point,omitempty"`
	LineString   *line         `xml:"LineString,omitempty"`
	Polygon      *polygon      `xml:"Polygon,omitempty"`
}

// My Maps imports Data fields as layer columns (e.g. Style by data column → Type).
type extendedData struct {
	Data []dataField `xml:"Data"`
}

type dataField struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value"`
}

type point struct {
	Coordinates string `xml:"coordinates"`
}

type line struct {
	Tessellate  int    `xml:"tessellate"`
	Coordinates string `xml:"coordinates"`
}

type polygon struct {
	OuterBoundaryIs outerBoundary `xml:"outerBoundaryIs"`
}

type outerBoundary struct {
	LinearRing linearRing `xml:"LinearRing"`
}

type linearRing struct {
	Coordinates string `xml:"coordinates"`
}

func iconRefsForExport(inline bool) (map[string]string, map[string][]byte, error) {
	refs := map[string]string{}
	files := map[string][]byte{}
	for _, t := range poi.AllTypes() {
		pngBytes, err := iconPNG(t)
		if err != nil {
			return nil, nil, err
		}
		key := string(t)
		files[key] = pngBytes
		if inline {
			refs[key] = dataURI(pngBytes)
		} else {
			refs[key] = "files/" + key + ".png"
		}
	}
	endPNG, err := routeEndPNG()
	if err != nil {
		return nil, nil, err
	}
	files["route-end"] = endPNG
	if inline {
		refs["route-end"] = dataURI(endPNG)
	} else {
		refs["route-end"] = "files/route-end.png"
	}
	for key := range campsiteIconBases {
		pngBytes, err := campsiteIconPNG(key)
		if err != nil {
			return nil, nil, err
		}
		files[key] = pngBytes
		if inline {
			refs[key] = dataURI(pngBytes)
		} else {
			refs[key] = "files/" + key + ".png"
		}
	}
	return refs, files, nil
}

func BuildKMZ(name string, pois []poi.POI, opts ExportOptions) ([]byte, error) {
	refs, files, err := iconRefsForExport(false)
	if err != nil {
		return nil, err
	}

	payload, err := buildKMLXML(name, pois, refs, opts)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	kmlFile, err := zw.Create("doc.kml")
	if err != nil {
		return nil, err
	}
	if _, err := kmlFile.Write([]byte(xml.Header)); err != nil {
		return nil, err
	}
	if _, err := kmlFile.Write(payload); err != nil {
		return nil, err
	}

	for key, pngBytes := range files {
		w, err := zw.Create("files/" + key + ".png")
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(pngBytes); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildKML returns a standalone KML document with icons inlined as data URIs.
func BuildKML(name string, pois []poi.POI, opts ExportOptions) ([]byte, error) {
	refs, _, err := iconRefsForExport(true)
	if err != nil {
		return nil, err
	}

	payload, err := buildKMLXML(name, pois, refs, opts)
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), payload...)
	return out, nil
}

func dataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func defaultLayers() []ExportLayer {
	return []ExportLayer{
		{Name: "Existing", Types: []string{"gym", "pokestop", "powerspot"}},
		{Name: "Routes", Types: []string{"route"}},
		{Name: "Areas", Types: []string{"outline"}},
		{Name: "Ranges", Types: []string{"range_wayfarer"}},
		{Name: "Campsite", Types: []string{"campsite_gym", "campsite_pokestop", "campsite_powerspot"}},
	}
}

func buildKMLXML(name string, pois []poi.POI, iconRefs map[string]string, opts ExportOptions) ([]byte, error) {
	if name == "" {
		name = "Pokémon GO map"
	}
	doc := document{Name: name}
	doc.Styles, doc.StyleMaps = sharedStyles(iconRefs)

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

	buckets := make([][]placemark, len(layers))
	for _, p := range pois {
		key := p.LayerTypeKey()
		idx, ok := typeToLayer[key]
		if !ok {
			continue
		}
		buckets[idx] = append(buckets[idx], placemarksForPOI(p)...)
	}

	if idx, ok := typeToLayer["outline"]; ok {
		areas := opts.Areas
		if len(areas) == 0 && len(opts.Outline) >= 3 {
			areas = []ExportArea{{Name: "Play Area", Ring: opts.Outline}}
		}
		seenOutlineStyles := map[string]bool{"outline": true}
		for _, a := range areas {
			if len(a.Ring) < 3 {
				continue
			}
			styleID := outlineStyleID(a.Color)
			if !seenOutlineStyles[styleID] {
				line := hexToKML(normalizeHexColor(a.Color), 0xff)
				fill := hexToKML(normalizeHexColor(a.Color), 0x40)
				doc.Styles, doc.StyleMaps = appendLinePolyStyleMap(doc.Styles, doc.StyleMaps, styleID, line, 3, fill)
				seenOutlineStyles[styleID] = true
			}
			buckets[idx] = append(buckets[idx], outlinePlacemark(a, styleID))
		}
	}

	for _, def := range exportRanges {
		idx, ok := typeToLayer[def.key]
		if !ok {
			continue
		}
		for _, p := range pois {
			if !rangeApplies(def, p) {
				continue
			}
			buckets[idx] = append(buckets[idx], circlePlacemark(p, def))
		}
	}

	for i, layer := range layers {
		items := buckets[i]
		if len(items) == 0 {
			continue
		}
		baseName := layer.Name
		if strings.TrimSpace(baseName) == "" {
			baseName = fmt.Sprintf("Layer %d", i+1)
		}
		for j := 0; j < len(items); j += featuresPerFolder {
			end := j + featuresPerFolder
			if end > len(items) {
				end = len(items)
			}
			fname := baseName
			if j > 0 {
				fname = fmt.Sprintf("%s %d", baseName, j/featuresPerFolder+1)
			}
			doc.Folder = append(doc.Folder, folder{
				Name:       fname,
				Placemarks: items[j:end],
			})
		}
	}

	return xml.MarshalIndent(kmlRoot{
		Xmlns:    "http://www.opengis.net/kml/2.2",
		Document: doc,
	}, "", "  ")
}

func sharedStyles(iconRefs map[string]string) ([]style, []styleMap) {
	var styles []style
	var maps []styleMap
	// My Maps expects StyleMap (normal/highlight) like its own export; shared Style-only
	// ids often collapse to a uniform layer style on import.
	centerHot := hotSpot{X: "0.5", Y: "0.5", XUnits: "fraction", YUnits: "fraction"}

	addIconStyleMap := func(id, href string) {
		for _, part := range []struct {
			suffix     string
			labelScale string
		}{
			{"-normal", "0"},
			{"-highlight", "1"},
		} {
			styles = append(styles, style{
				ID: id + part.suffix,
				IconStyle: &iconStyle{
					Scale:   iconScale,
					Icon:    icon{Href: href},
					HotSpot: centerHot,
				},
				LabelStyle: &labelStyle{Scale: part.labelScale},
			})
		}
		maps = append(maps, styleMap{
			ID: id,
			Pair: []stylePair{
				{Key: "normal", StyleURL: "#" + id + "-normal"},
				{Key: "highlight", StyleURL: "#" + id + "-highlight"},
			},
		})
	}

	addLinePolyStyleMap := func(id, lineColor string, lineWidth int, polyColor string) {
		styles, maps = appendLinePolyStyleMap(styles, maps, id, lineColor, lineWidth, polyColor)
	}

	for _, t := range poi.AllTypes() {
		id := string(t)
		addIconStyleMap(id, iconRefs[id])
	}
	for key := range campsiteIconBases {
		addIconStyleMap(key, iconRefs[key])
	}

	addLinePolyStyleMap("route-line", routeLineKML, 4, "")
	addIconStyleMap("route-end", iconRefs["route-end"])
	addLinePolyStyleMap("outline", "ff8cff5b", 3, "405b8cff")

	for _, def := range exportRanges {
		addLinePolyStyleMap(def.key, hexToKML(def.color, 0xff), 2, hexToKML(def.color, 0x40))
	}

	return styles, maps
}

func typeData(typeValue string) *extendedData {
	return &extendedData{Data: []dataField{{Name: "Type", Value: typeValue}}}
}

// typeColumn is the My Maps “Type” column value for Style by data column.
func typeColumn(p poi.POI) string {
	if p.IsCampsite() {
		switch p.Type {
		case poi.TypeGym, poi.TypeSuperMegaGym:
			return "Campsite Gym"
		case poi.TypePokeStop:
			return "Campsite PokéStop"
		case poi.TypePowerspot:
			return "Campsite Powerspot"
		}
	}
	switch p.Type {
	case poi.TypeGym:
		return "Gym"
	case poi.TypeSuperMegaGym:
		return "Super Mega Gym"
	case poi.TypePokeStop:
		return "PokéStop"
	case poi.TypePowerspot:
		if p.IsInactivePowerspot() {
			return "Powerspot · Inactive"
		}
		return "Powerspot"
	case poi.TypeRoute:
		return "Route"
	default:
		return string(p.Type)
	}
}

func poiDescription(p poi.POI, typeValue string) string {
	return fmt.Sprintf("%s · %.6f, %.6f · Type: %s", p.Name, p.Lat, p.Lng, typeValue)
}

// placemarksForPOI returns one or more placemarks. Routes keep start, path, and end together
// (caller places them in the same folder). Styles are shared by POI type in the KML file;
// My Maps typically imports them as individual styles per feature — that's fine.
func placemarksForPOI(p poi.POI) []placemark {
	styleID := p.LayerTypeKey()
	if p.Type == poi.TypeRoute && len(p.Path) > 1 {
		coords := make([]string, 0, len(p.Path))
		for _, pt := range p.Path {
			coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
		}
		start := p.Path[0]
		endPt := p.Path[len(p.Path)-1]
		startType, pathType, endType := "Route", "Route path", "Route end"
		return []placemark{
			{
				Name:         p.Name,
				Description:  poiDescription(p, startType),
				StyleURL:     "#route",
				ExtendedData: typeData(startType),
				Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", start[1], start[0])},
			},
			{
				Name:         p.Name,
				Description:  poiDescription(p, pathType),
				StyleURL:     "#route-line",
				ExtendedData: typeData(pathType),
				LineString:   &line{Tessellate: 1, Coordinates: strings.Join(coords, " ")},
			},
			{
				Name:         p.Name + " (end)",
				Description:  poiDescription(p, endType),
				StyleURL:     "#route-end",
				ExtendedData: typeData(endType),
				Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", endPt[1], endPt[0])},
			},
		}
	}
	t := typeColumn(p)
	return []placemark{{
		Name:         p.Name,
		Description:  poiDescription(p, t),
		StyleURL:     "#" + styleID,
		ExtendedData: typeData(t),
		Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", p.Lng, p.Lat)},
	}}
}

func appendLinePolyStyleMap(styles []style, maps []styleMap, id, lineColor string, lineWidth int, polyColor string) ([]style, []styleMap) {
	for _, suffix := range []string{"-normal", "-highlight"} {
		s := style{
			ID:        id + suffix,
			LineStyle: &lineStyle{Color: lineColor, Width: lineWidth},
		}
		if polyColor != "" {
			s.PolyStyle = &polyStyle{Color: polyColor, Fill: 1, Outline: 1}
		}
		styles = append(styles, s)
	}
	maps = append(maps, styleMap{
		ID: id,
		Pair: []stylePair{
			{Key: "normal", StyleURL: "#" + id + "-normal"},
			{Key: "highlight", StyleURL: "#" + id + "-highlight"},
		},
	})
	return styles, maps
}

const defaultOutlineHex = "#5b8cff"

func normalizeHexColor(hex string) string {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return defaultOutlineHex
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return defaultOutlineHex
		}
	}
	return "#" + strings.ToLower(h)
}

// outlineStyleID returns a KML style id for an area color (#rrggbb → outline_rrggbb).
func outlineStyleID(hex string) string {
	h := strings.TrimPrefix(normalizeHexColor(hex), "#")
	if h == "5b8cff" {
		return "outline" // shared default style from sharedStyles
	}
	return "outline_" + h
}

func outlinePlacemark(area ExportArea, styleID string) placemark {
	ring := area.Ring
	coords := make([]string, 0, len(ring)+1)
	for _, pt := range ring {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
	}
	first := ring[0]
	last := ring[len(ring)-1]
	if first[0] != last[0] || first[1] != last[1] {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", first[1], first[0]))
	}
	name := strings.TrimSpace(area.Name)
	if name == "" {
		name = "Area"
	}
	t := name
	if styleID == "" {
		styleID = outlineStyleID(area.Color)
	}
	return placemark{
		Name:         name,
		Description:  "Map area · Type: " + t,
		StyleURL:     "#" + styleID,
		ExtendedData: typeData(t),
		Polygon: &polygon{
			OuterBoundaryIs: outerBoundary{
				LinearRing: linearRing{Coordinates: strings.Join(coords, " ")},
			},
		},
	}
}
