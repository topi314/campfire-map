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
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	StyleURL    string   `xml:"styleUrl"`
	Point       *point   `xml:"Point,omitempty"`
	LineString  *line    `xml:"LineString,omitempty"`
	Polygon     *polygon `xml:"Polygon,omitempty"`
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
		{Name: "Play Area", Types: []string{"outline"}},
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

	if len(opts.Outline) >= 3 {
		if idx, ok := typeToLayer["outline"]; ok {
			buckets[idx] = append(buckets[idx], outlinePlacemark(opts.Outline))
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
	// My Maps imports StyleMap poorly for custom icons; use a single shared Style per type.
	var maps []styleMap
	centerHot := hotSpot{X: "0.5", Y: "0.5", XUnits: "fraction", YUnits: "fraction"}

	addIconStyle := func(id, href string) {
		styles = append(styles, style{
			ID: id,
			IconStyle: &iconStyle{
				Scale:   iconScale,
				Icon:    icon{Href: href},
				HotSpot: centerHot,
			},
			LabelStyle: &labelStyle{Scale: "0"},
		})
	}

	for _, t := range poi.AllTypes() {
		id := string(t)
		addIconStyle(id, iconRefs[id])
	}
	for key := range campsiteIconBases {
		addIconStyle(key, iconRefs[key])
	}

	styles = append(styles, style{
		ID:        "route-line",
		LineStyle: &lineStyle{Color: routeLineKML, Width: 4},
	})

	addIconStyle("route-end", iconRefs["route-end"])

	styles = append(styles, style{
		ID:        "outline",
		LineStyle: &lineStyle{Color: "ff8cff5b", Width: 3},
		PolyStyle: &polyStyle{Color: "405b8cff", Fill: 1, Outline: 1},
	})

	return styles, maps
}

// placemarksForPOI returns one or more placemarks. Routes keep start, path, and end together
// (caller places them in the same folder). Styles are shared by POI type in the KML file;
// My Maps typically imports them as individual styles per feature — that's fine.
func placemarksForPOI(p poi.POI) []placemark {
	desc := fmt.Sprintf("%s · %.6f, %.6f", p.Type.Label(), p.Lat, p.Lng)
	if p.IsCampsite() {
		desc = "Campsite " + desc
	}
	styleID := p.LayerTypeKey()
	if p.Type == poi.TypeRoute && len(p.Path) > 1 {
		coords := make([]string, 0, len(p.Path))
		for _, pt := range p.Path {
			coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
		}
		start := p.Path[0]
		endPt := p.Path[len(p.Path)-1]
		return []placemark{
			{
				Name:        p.Name,
				Description: desc,
				StyleURL:    "#route",
				Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", start[1], start[0])},
			},
			{
				Name:        p.Name,
				Description: desc,
				StyleURL:    "#route-line",
				LineString:  &line{Tessellate: 1, Coordinates: strings.Join(coords, " ")},
			},
			{
				Name:        p.Name + " (end)",
				Description: desc,
				StyleURL:    "#route-end",
				Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", endPt[1], endPt[0])},
			},
		}
	}
	return []placemark{{
		Name:        p.Name,
		Description: desc,
		StyleURL:    "#" + styleID,
		Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", p.Lng, p.Lat)},
	}}
}

func outlinePlacemark(ring [][2]float64) placemark {
	coords := make([]string, 0, len(ring)+1)
	for _, pt := range ring {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
	}
	first := ring[0]
	last := ring[len(ring)-1]
	if first[0] != last[0] || first[1] != last[1] {
		coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", first[1], first[0]))
	}
	return placemark{
		Name:        "Draw area",
		Description: "Planned campsite area",
		StyleURL:    "#outline",
		Polygon: &polygon{
			OuterBoundaryIs: outerBoundary{
				LinearRing: linearRing{Coordinates: strings.Join(coords, " ")},
			},
		},
	}
}
