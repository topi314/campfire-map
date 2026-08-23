package campfire

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/topi314/campfire-export/server/poi"
)

func parseMapObjects(data json.RawMessage) ([]poi.POI, error) {
	var tile tileResponse
	if err := json.Unmarshal(data, &tile); err != nil {
		return nil, err
	}
	if tile.RealityChannelMapObjectsByS2Cells == nil {
		return nil, nil
	}

	byID := map[string]int{}
	var out []poi.POI
	for _, cell := range tile.RealityChannelMapObjectsByS2Cells.MapObjectsByS2CellsAndTypes {
		for _, group := range cell.MapObjectsByType {
			for _, obj := range group.MapObjects {
				if p, ok := obj.toPOI(group.Type); ok {
					addPOI(p, &out, byID)
				}
			}
		}
	}
	return out, nil
}

func addPOI(p poi.POI, out *[]poi.POI, byID map[string]int) {
	if p.ID == "" {
		*out = append(*out, p)
		return
	}
	if i, ok := byID[p.ID]; ok {
		(*out)[i] = mergePOI((*out)[i], p)
		return
	}
	byID[p.ID] = len(*out)
	*out = append(*out, p)
}

func (m mapObject) toPOI(groupType string) (poi.POI, bool) {
	groupType = firstNonEmpty(groupType, m.MapObjectType)
	if m.PgoGym != nil {
		if p, ok := m.PgoGym.toPOI(groupType); ok {
			return p, true
		}
	}
	if m.PgoPokestop != nil {
		if p, ok := m.PgoPokestop.toPOI(); ok {
			return p, true
		}
	}
	if m.PgoPowerspot != nil {
		if p, ok := m.PgoPowerspot.toPOI(); ok {
			return p, true
		}
	}
	if m.PgoRoute != nil {
		if p, ok := m.PgoRoute.toPOI(); ok {
			return p, true
		}
	}
	return poi.POI{}, false
}

func (g *pgoGym) toPOI(groupType string) (poi.POI, bool) {
	lat, lng, ok := g.Location.ok()
	if !ok {
		return poi.POI{}, false
	}
	name := g.Name
	if name == "" {
		name = fallbackName(poi.TypeGym)
	}
	return poi.POI{
		ID:                coordID("gym", lat, lng),
		Type:              poi.TypeGym,
		Name:              name,
		Icon:              g.ImageURL,
		Lat:               lat,
		Lng:               lng,
		SuperMegaEligible: g.IsMegaEnhancedEligible.truthy() || looksLikeSuperMegaType(groupType),
	}, true
}

func (s *pgoPokestop) toPOI() (poi.POI, bool) {
	lat, lng, ok := s.Location.ok()
	if !ok {
		return poi.POI{}, false
	}
	name := s.Name
	if name == "" {
		name = fallbackName(poi.TypePokeStop)
	}
	return poi.POI{
		ID:   coordID("stop", lat, lng),
		Type: poi.TypePokeStop,
		Name: name,
		Icon: s.ImageURL,
		Lat:  lat,
		Lng:  lng,
	}, true
}

func (s *pgoPowerspot) toPOI() (poi.POI, bool) {
	lat, lng, ok := s.Location.ok()
	if !ok {
		return poi.POI{}, false
	}
	name := s.Name
	if name == "" {
		name = fallbackName(poi.TypePowerspot)
	}
	return poi.POI{ID: coordID("powerspot", lat, lng), Type: poi.TypePowerspot, Name: name, Lat: lat, Lng: lng}, true
}

func (r *pgoRoute) toPOI() (poi.POI, bool) {
	path := r.path()
	if len(path) == 0 {
		return poi.POI{}, false
	}
	id := r.ID
	if id == "" {
		id = coordID("route", path[0][0], path[0][1])
	}
	name := r.Name
	if name == "" {
		name = fallbackName(poi.TypeRoute)
	}
	icon := ""
	if r.StartPoi != nil {
		icon = r.StartPoi.ImageURL
	}
	return poi.POI{
		ID:   id,
		Type: poi.TypeRoute,
		Name: name,
		Icon: icon,
		Lat:  path[0][0],
		Lng:  path[0][1],
		Path: path,
	}, true
}

func (r *pgoRoute) path() [][2]float64 {
	if len(r.LocationList) > 0 {
		path := make([][2]float64, 0, len(r.LocationList))
		for i := range r.LocationList {
			lat, lng, ok := r.LocationList[i].ok()
			if ok {
				path = append(path, [2]float64{lat, lng})
			}
		}
		if len(path) > 0 {
			return path
		}
	}
	var path [][2]float64
	for _, p := range []*routePoi{r.StartPoi, r.EndPoi} {
		if p == nil {
			continue
		}
		lat, lng, ok := p.Location.ok()
		if ok {
			path = append(path, [2]float64{lat, lng})
		}
	}
	return path
}

func mergePOI(cur, upd poi.POI) poi.POI {
	if upd.Name != "" && upd.Name != fallbackName(cur.Type) {
		cur.Name = upd.Name
	}
	if upd.Icon != "" {
		cur.Icon = upd.Icon
	}
	if len(upd.Path) > len(cur.Path) {
		cur.Path = upd.Path
	}
	if upd.SuperMegaEligible || upd.Type == poi.TypeSuperMegaGym || cur.Type == poi.TypeSuperMegaGym {
		cur.SuperMegaEligible = true
	}
	if cur.Type == poi.TypeSuperMegaGym {
		cur.Type = poi.TypeGym
	}
	return cur
}

func looksLikeSuperMegaType(s string) bool {
	u := strings.ToUpper(s)
	if u == "" {
		return false
	}
	return strings.Contains(u, "SUPER_MEGA") ||
		strings.Contains(u, "MEGA_ENHANCED") ||
		(strings.Contains(u, "MEGA") && strings.Contains(u, "GYM"))
}

func fallbackName(t poi.Type) string {
	switch t {
	case poi.TypeGym:
		return "Gym"
	case poi.TypeSuperMegaGym:
		return "Super Mega Gym"
	case poi.TypePokeStop:
		return "PokéStop"
	case poi.TypePowerspot:
		return "Powerspot"
	case poi.TypeRoute:
		return "Route"
	default:
		return "POI"
	}
}

func coordID(prefix string, lat, lng float64) string {
	return prefix + ":" + strconv.FormatFloat(lat, 'f', 6, 64) + "," + strconv.FormatFloat(lng, 'f', 6, 64)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
