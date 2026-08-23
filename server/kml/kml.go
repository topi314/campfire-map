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

// Folder keys so each My Maps layer can use Uniform style (one styleUrl per folder).
const (
	folderRoutePaths = "route-paths"
	folderRouteEnds  = "route-ends"
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
	ID         string      `xml:"id,attr"`
	IconStyle  *iconStyle  `xml:"IconStyle,omitempty"`
	LabelStyle *labelStyle `xml:"LabelStyle,omitempty"`
	LineStyle  *lineStyle  `xml:"LineStyle,omitempty"`
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
	Name        string `xml:"name"`
	Description string `xml:"description"`
	StyleURL    string `xml:"styleUrl"`
	Point       *point `xml:"Point,omitempty"`
	LineString  *line  `xml:"LineString,omitempty"`
}

type point struct {
	Coordinates string `xml:"coordinates"`
}

type line struct {
	Tessellate  int    `xml:"tessellate"`
	Coordinates string `xml:"coordinates"`
}

func BuildKMZ(name string, pois []poi.POI) ([]byte, error) {
	refs := map[string]string{}
	for _, t := range poi.AllTypes() {
		refs[string(t)] = "files/" + string(t) + ".png"
	}
	refs["route-end"] = "files/route-end.png"

	payload, err := buildKMLXML(name, pois, refs)
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

	for _, t := range poi.AllTypes() {
		pngBytes, err := iconPNG(t)
		if err != nil {
			return nil, err
		}
		w, err := zw.Create("files/" + string(t) + ".png")
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(pngBytes); err != nil {
			return nil, err
		}
	}
	endPNG, err := routeEndPNG()
	if err != nil {
		return nil, err
	}
	w, err := zw.Create("files/route-end.png")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(endPNG); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildKML returns a standalone KML document with icons inlined as data URIs.
func BuildKML(name string, pois []poi.POI) ([]byte, error) {
	refs := map[string]string{}
	for _, t := range poi.AllTypes() {
		pngBytes, err := iconPNG(t)
		if err != nil {
			return nil, err
		}
		refs[string(t)] = dataURI(pngBytes)
	}
	endPNG, err := routeEndPNG()
	if err != nil {
		return nil, err
	}
	refs["route-end"] = dataURI(endPNG)

	payload, err := buildKMLXML(name, pois, refs)
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), payload...)
	return out, nil
}

func dataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func buildKMLXML(name string, pois []poi.POI, iconRefs map[string]string) ([]byte, error) {
	if name == "" {
		name = "Pokémon GO map"
	}
	doc := document{Name: name}
	doc.Styles, doc.StyleMaps = sharedStyles(iconRefs)

	// One styleUrl per folder so My Maps imports layers as Uniform style.
	buckets := map[string][]placemark{}
	for _, p := range pois {
		for key, m := range placemarksByFolder(p) {
			buckets[key] = append(buckets[key], m)
		}
	}
	for _, key := range folderOrder() {
		items := buckets[key]
		if len(items) == 0 {
			continue
		}
		baseName := folderLabel(key)
		for i := 0; i < len(items); i += featuresPerFolder {
			end := i + featuresPerFolder
			if end > len(items) {
				end = len(items)
			}
			fname := baseName
			if i > 0 {
				fname = fmt.Sprintf("%s %d", baseName, i/featuresPerFolder+1)
			}
			doc.Folder = append(doc.Folder, folder{
				Name:       fname,
				Placemarks: items[i:end],
			})
		}
	}

	return xml.MarshalIndent(kmlRoot{
		Xmlns:    "http://www.opengis.net/kml/2.2",
		Document: doc,
	}, "", "  ")
}

func folderOrder() []string {
	keys := make([]string, 0, len(poi.FolderTypes())+2)
	for _, t := range poi.FolderTypes() {
		keys = append(keys, string(t))
	}
	keys = append(keys, folderRoutePaths, folderRouteEnds)
	return keys
}

func folderLabel(key string) string {
	switch key {
	case folderRoutePaths:
		return "Route paths"
	case folderRouteEnds:
		return "Route ends"
	default:
		return poi.Type(key).Label()
	}
}

func sharedStyles(iconRefs map[string]string) ([]style, []styleMap) {
	var styles []style
	var maps []styleMap
	centerHot := hotSpot{X: "0.5", Y: "0.5", XUnits: "fraction", YUnits: "fraction"}

	iconStyleMap := func(id, href string, showLabelOnHighlight bool) {
		normalLabel := "0"
		highlightLabel := "0"
		if showLabelOnHighlight {
			highlightLabel = "1"
		}
		styles = append(styles,
			style{
				ID: id + "-normal",
				IconStyle: &iconStyle{
					Scale:   iconScale,
					Icon:    icon{Href: href},
					HotSpot: centerHot,
				},
				LabelStyle: &labelStyle{Scale: normalLabel},
			},
			style{
				ID: id + "-highlight",
				IconStyle: &iconStyle{
					Scale:   iconScale,
					Icon:    icon{Href: href},
					HotSpot: centerHot,
				},
				LabelStyle: &labelStyle{Scale: highlightLabel},
			},
		)
		maps = append(maps, styleMap{
			ID: id,
			Pair: []stylePair{
				{Key: "normal", StyleURL: "#" + id + "-normal"},
				{Key: "highlight", StyleURL: "#" + id + "-highlight"},
			},
		})
	}

	for _, t := range poi.AllTypes() {
		id := string(t)
		iconStyleMap(id, iconRefs[id], true)
	}

	styles = append(styles,
		style{
			ID:        "route-line-normal",
			LineStyle: &lineStyle{Color: routeLineKML, Width: 4},
		},
		style{
			ID:        "route-line-highlight",
			LineStyle: &lineStyle{Color: routeLineKML, Width: 6},
		},
	)
	maps = append(maps, styleMap{
		ID: "route-line",
		Pair: []stylePair{
			{Key: "normal", StyleURL: "#route-line-normal"},
			{Key: "highlight", StyleURL: "#route-line-highlight"},
		},
	})

	iconStyleMap("route-end", iconRefs["route-end"], false)
	return styles, maps
}

// placemarksByFolder returns placemarks keyed by folder. Each folder uses a single styleUrl
// so My Maps can apply Uniform style per layer.
func placemarksByFolder(p poi.POI) map[string]placemark {
	desc := fmt.Sprintf("%s<br/>%.6f, %.6f", p.Name, p.Lat, p.Lng)
	if p.Type == poi.TypeRoute && len(p.Path) > 1 {
		coords := make([]string, 0, len(p.Path))
		for _, pt := range p.Path {
			coords = append(coords, fmt.Sprintf("%.7f,%.7f,0", pt[1], pt[0]))
		}
		start := p.Path[0]
		endPt := p.Path[len(p.Path)-1]
		return map[string]placemark{
			string(poi.TypeRoute): {
				Name:        p.Name,
				Description: desc,
				StyleURL:    "#route",
				Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", start[1], start[0])},
			},
			folderRoutePaths: {
				Name:        p.Name,
				Description: desc,
				StyleURL:    "#route-line",
				LineString:  &line{Tessellate: 1, Coordinates: strings.Join(coords, " ")},
			},
			folderRouteEnds: {
				Name:        p.Name + " (end)",
				Description: desc,
				StyleURL:    "#route-end",
				Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", endPt[1], endPt[0])},
			},
		}
	}
	return map[string]placemark{
		string(p.Type.FolderType()): {
			Name:        p.Name,
			Description: desc,
			StyleURL:    "#" + string(p.Type.FolderType()),
			Point:       &point{Coordinates: fmt.Sprintf("%.7f,%.7f,0", p.Lng, p.Lat)},
		},
	}
}
