package kml

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math"
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
	Color string  `xml:"color"`
	Width float64 `xml:"width"`
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
	StyleURL     string        `xml:"styleUrl,omitempty"`
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
	Tessellate  *int   `xml:"tessellate,omitempty"`
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
		{Name: "Existing", Types: []string{"gym", "pokestop", "powerspot", "route"}},
		{Name: "Areas", Types: []string{"outline"}},
		{Name: "Campsite Routes", Types: []string{"drawn_route"}},
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
		// Match My Maps export: shared poly-RRGGBB-3000-64 StyleMaps + styleUrl
		// (native My Maps KMZ re-imports with Individual colors this way).
		seenOutlineStyles := map[string]bool{}
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

	if idx, ok := typeToLayer["drawn_route"]; ok {
		seenRouteStyles := map[string]bool{drawnRouteLineStyleID(defaultOutlineHex): true}
		for _, rt := range opts.Routes {
			if len(rt.Path) < 2 {
				continue
			}
			lineStyleID := drawnRouteLineStyleID(rt.Color)
			if !seenRouteStyles[lineStyleID] {
				line := hexToKML(normalizeHexColor(rt.Color), 0xff)
				doc.Styles, doc.StyleMaps = appendLinePolyStyleMap(doc.Styles, doc.StyleMaps, lineStyleID, line, 4, "")
				seenRouteStyles[lineStyleID] = true
			}
			buckets[idx] = append(buckets[idx], drawnRoutePlacemarks(rt, lineStyleID)...)
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

	addLinePolyStyleMap := func(id, lineColor string, lineWidth float64, polyColor string) {
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
	// My Maps drawn-route lines: line-RRGGBB-4000 (width 4 → highlight 6).
	outlineLine := hexToKML(defaultOutlineHex, 0xff)
	addLinePolyStyleMap(drawnRouteLineStyleID(defaultOutlineHex), outlineLine, 4, "")

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

// placemarksForPOI returns one or more placemarks. Routes keep start + path together
// (and end when the path is open). Closed loops omit the end flag — start covers both ends.
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
		out := []placemark{
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
		}
		if !routeEndpointsConnected(p.Path) {
			out = append(out, placemark{
				Name:         p.Name + " (end)",
				Description:  poiDescription(p, endType),
				StyleURL:     "#route-end",
				ExtendedData: typeData(endType),
				Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", endPt[1], endPt[0])},
			})
		}
		return out
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

// appendLinePolyStyleMap mirrors My Maps exports: normal width + highlight at 1.5×
// (e.g. poly width 3 → highlight 4.5, line width 4 → highlight 6).
func appendLinePolyStyleMap(styles []style, maps []styleMap, id, lineColor string, lineWidth float64, polyColor string) ([]style, []styleMap) {
	for _, part := range []struct {
		suffix string
		width  float64
	}{
		{"-normal", lineWidth},
		{"-highlight", lineWidth * 1.5},
	} {
		s := style{
			ID:        id + part.suffix,
			LineStyle: &lineStyle{Color: lineColor, Width: part.width},
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

// outlineStyleID matches My Maps polygon naming: poly-RRGGBB-3000-64
// (width 3.0 → 3000, fill alpha 0x40 → 64).
func outlineStyleID(hex string) string {
	h := strings.ToUpper(strings.TrimPrefix(normalizeHexColor(hex), "#"))
	return "poly-" + h + "-3000-64"
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
	desc := "Map area · Type: " + t
	tess := 1
	return placemark{
		Name:        name,
		Description: desc,
		StyleURL:    "#" + styleID,
		ExtendedData: &extendedData{Data: []dataField{
			{Name: "description", Value: desc},
			{Name: "Type", Value: t},
		}},
		Polygon: &polygon{
			OuterBoundaryIs: outerBoundary{
				LinearRing: linearRing{Tessellate: &tess, Coordinates: strings.Join(coords, " ")},
			},
		},
	}
}

// drawnRouteLineStyleID matches My Maps line naming: line-RRGGBB-4000 (width 4.0).
func drawnRouteLineStyleID(hex string) string {
	h := strings.ToUpper(strings.TrimPrefix(normalizeHexColor(hex), "#"))
	return "line-" + h + "-4000"
}

func routeExtendedData(desc, typeValue string) *extendedData {
	return &extendedData{Data: []dataField{
		{Name: "description", Value: desc},
		{Name: "Type", Value: typeValue},
	}}
}

// drawnRoutePlacemarks emits start + path, and end when the path is open.
// Closed loops omit the end flag. ExtendedData Type uses the route name.
func drawnRoutePlacemarks(rt ExportRoute, lineStyleID string) []placemark {
	name := strings.TrimSpace(rt.Name)
	if name == "" {
		name = "Route"
	}
	coords := make([]string, 0, len(rt.Path))
	for _, pt := range rt.Path {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
	}
	start := rt.Path[0]
	endPt := rt.Path[len(rt.Path)-1]
	if lineStyleID == "" {
		lineStyleID = drawnRouteLineStyleID(rt.Color)
	}
	desc := fmt.Sprintf("%s · %.6f, %.6f · Type: %s", name, start[0], start[1], name)
	ext := routeExtendedData(desc, name)
	out := []placemark{
		{
			Name:         name,
			Description:  desc,
			StyleURL:     "#route",
			ExtendedData: ext,
			Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", start[1], start[0])},
		},
		{
			Name:         name,
			Description:  desc,
			StyleURL:     "#" + lineStyleID,
			ExtendedData: ext,
			LineString:   &line{Tessellate: 1, Coordinates: strings.Join(coords, " ")},
		},
	}
	if !routeEndpointsConnected(rt.Path) {
		out = append(out, placemark{
			Name:         name + " (end)",
			Description:  desc,
			StyleURL:     "#route-end",
			ExtendedData: ext,
			Point:        &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", endPt[1], endPt[0])},
		})
	}
	return out
}

// routeEndpointsConnected reports whether start and end are the same point (closed loop).
// Threshold matches the map preview (~4 m).
func routeEndpointsConnected(path [][2]float64) bool {
	if len(path) < 2 {
		return false
	}
	a, b := path[0], path[len(path)-1]
	const maxM = 4.0
	return haversineMeters(a[0], a[1], b[0], b[1]) <= maxM
}

func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0
	toRad := math.Pi / 180
	φ1, φ2 := lat1*toRad, lat2*toRad
	dφ := (lat2 - lat1) * toRad
	dλ := (lng2 - lng1) * toRad
	s := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return 2 * R * math.Asin(math.Min(1, math.Sqrt(s)))
}
