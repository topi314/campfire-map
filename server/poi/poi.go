package poi

import "errors"

var errInvalidBBox = errors.New("bbox must be minLat,minLng,maxLat,maxLng")

type Type string

const (
	TypeGym          Type = "gym"
	TypeSuperMegaGym Type = "super_mega_gym"
	TypePokeStop     Type = "pokestop"
	TypePowerspot    Type = "powerspot"
	TypeRoute        Type = "route"
)

func AllTypes() []Type {
	return []Type{TypeGym, TypeSuperMegaGym, TypePokeStop, TypePowerspot, TypeRoute}
}

func FolderTypes() []Type {
	return []Type{TypeGym, TypeSuperMegaGym, TypePokeStop, TypePowerspot, TypeRoute}
}

func (t Type) FolderType() Type {
	return t
}

func (t Type) Label() string {
	switch t {
	case TypeGym:
		return "Gyms"
	case TypeSuperMegaGym:
		return "Super Mega Gyms"
	case TypePokeStop:
		return "PokéStops"
	case TypePowerspot:
		return "Powerspot"
	case TypeRoute:
		return "Routes"
	default:
		return string(t)
	}
}

func ParseType(s string) (Type, bool) {
	switch Type(s) {
	case TypeGym, TypeSuperMegaGym, TypePokeStop, TypePowerspot, TypeRoute:
		return Type(s), true
	case "dmax", "dynaspot":
		return TypePowerspot, true
	default:
		return "", false
	}
}

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type POI struct {
	ID                string       `json:"id"`
	Type              Type         `json:"type"`
	Name              string       `json:"name"`
	Icon              string       `json:"icon"`
	Lat               float64      `json:"lat"`
	Lng               float64      `json:"lng"`
	Path              [][2]float64 `json:"path,omitempty"`
	SuperMegaEligible bool         `json:"superMegaEligible,omitempty"`
	// Status is set for Wayfarer powerspots ("active" / "inactive"); empty for GraphQL sources.
	Status string `json:"status,omitempty"`
}

func (p POI) IsInactivePowerspot() bool {
	return p.Type == TypePowerspot && p.Status == StatusInactive
}

func (p *POI) Normalize() {
	if p.Type == "dmax" || p.Type == "dynaspot" {
		p.Type = TypePowerspot
	}
}

func (p POI) IsGym() bool {
	return p.Type == TypeGym || p.Type == TypeSuperMegaGym
}

func (p POI) IsSuperMegaGym() bool {
	return p.SuperMegaEligible || p.Type == TypeSuperMegaGym
}

func (p POI) MatchesType(t Type) bool {
	if t == "dmax" || t == "dynaspot" {
		t = TypePowerspot
	}
	switch t {
	case TypeGym:
		return p.IsGym()
	case TypeSuperMegaGym:
		return p.IsSuperMegaGym()
	case TypePowerspot:
		return p.Type == TypePowerspot || p.Type == "dynaspot" || p.Type == "dmax"
	default:
		return p.Type == t
	}
}

type BBox struct {
	MinLat float64 `json:"minLat"`
	MinLng float64 `json:"minLng"`
	MaxLat float64 `json:"maxLat"`
	MaxLng float64 `json:"maxLng"`
}

func (b BBox) Valid() error {
	if b.MinLat < -90 || b.MaxLat > 90 || b.MinLng < -180 || b.MaxLng > 180 {
		return errInvalidBBox
	}
	if b.MinLat >= b.MaxLat || b.MinLng >= b.MaxLng {
		return errInvalidBBox
	}
	return nil
}

func (b BBox) SpanTooLarge(maxSpan float64) bool {
	if maxSpan <= 0 {
		return false
	}
	return (b.MaxLat-b.MinLat) > maxSpan || (b.MaxLng-b.MinLng) > maxSpan
}

// ClampSpan shrinks the bbox around its center so neither span exceeds maxSpan.
func (b BBox) ClampSpan(maxSpan float64) BBox {
	if maxSpan <= 0 || !b.SpanTooLarge(maxSpan) {
		return b
	}
	clat, clng := b.Center()
	half := maxSpan / 2
	out := BBox{
		MinLat: clat - half,
		MaxLat: clat + half,
		MinLng: clng - half,
		MaxLng: clng + half,
	}
	if out.MinLat < -90 {
		out.MaxLat += -90 - out.MinLat
		out.MinLat = -90
	}
	if out.MaxLat > 90 {
		out.MinLat -= out.MaxLat - 90
		out.MaxLat = 90
	}
	if out.MinLat < -90 {
		out.MinLat = -90
	}
	if out.MinLng < -180 {
		out.MaxLng += -180 - out.MinLng
		out.MinLng = -180
	}
	if out.MaxLng > 180 {
		out.MinLng -= out.MaxLng - 180
		out.MaxLng = 180
	}
	if out.MinLng < -180 {
		out.MinLng = -180
	}
	return out
}

func (b BBox) Contains(lat, lng float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng
}

func (b BBox) Center() (lat, lng float64) {
	return (b.MinLat + b.MaxLat) / 2, (b.MinLng + b.MaxLng) / 2
}
