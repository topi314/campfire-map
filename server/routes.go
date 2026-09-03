package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/topi314/campfire-export/server/campfire"
	"github.com/topi314/campfire-export/server/kml"
	"github.com/topi314/campfire-export/server/poi"
	"github.com/topi314/campfire-export/server/s2cells"
)

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) clientConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"cartoApiKey": s.cfg.Basemaps.CartoKey()})
}

func (s *Server) getPOIs(w http.ResponseWriter, r *http.Request) {
	bbox, err := parseBBox(r.URL.Query().Get("bbox"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bbox.Valid(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bbox = bbox.ClampSpan(s.cfg.Limits.MaxBBoxSpan)
	types := parseTypes(r.URL.Query().Get("types"))
	fetchCells, level, err := s2cells.CoverFirstFit(bbox, s.cfg.Limits.MaxCells, s2cells.Level15)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	overlay, err := s2cells.Overlay(bbox)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cellIDs := make([]string, len(fetchCells))
	for i, c := range fetchCells {
		cellIDs[i] = c.ID
	}
	tok := bearerToken(r)
	key := strings.Join(cellIDs, ",") + ":" + campfire.CacheKeySuffix(tok)
	var pois []poi.POI
	if raw, ok := s.cache.Get(key); ok {
		_ = json.Unmarshal(raw, &pois)
	} else {
		pois, err = s.client.Fetch(campfire.WithToken(r.Context(), tok), cellIDs, level)
		if err != nil {
			slog.Error("fetch pois", slog.Any("err", err))
			http.Error(w, "failed to fetch map data: "+err.Error(), http.StatusBadGateway)
			return
		}
		if raw, err := json.Marshal(pois); err == nil {
			s.cache.Set(key, raw)
		}
	}

	filtered := make([]poi.POI, 0, len(pois))
	for _, p := range pois {
		p.Normalize()
		if !bbox.Contains(p.Lat, p.Lng) {
			continue
		}
		if len(types) > 0 && !matchesAnyType(p, types) {
			continue
		}
		filtered = append(filtered, p)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pois":      filtered,
		"count":     len(filtered),
		"cellCount": len(fetchCells),
		"cellLevel": level,
		"cells":     overlay,
	})
}

type exportReq struct {
	Name   string    `json:"name"`
	Format string    `json:"format"`
	POIs   []poi.POI `json:"pois"`
}

func (s *Server) exportKMZ(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	var req exportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.POIs) == 0 {
		http.Error(w, "no pois selected", http.StatusBadRequest)
		return
	}
	if s.cfg.Limits.MaxExportPOIs > 0 && len(req.POIs) > s.cfg.Limits.MaxExportPOIs {
		http.Error(w, "too many POIs selected", http.StatusBadRequest)
		return
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "kmz"
	}
	var (
		data     []byte
		err      error
		filename string
		ctype    string
	)
	switch format {
	case "kml":
		data, err = kml.BuildKML(req.Name, req.POIs)
		filename = "pogo-export.kml"
		ctype = "application/vnd.google-earth.kml+xml"
	case "kmz":
		data, err = kml.BuildKMZ(req.Name, req.POIs)
		filename = "pogo-export.kmz"
		ctype = "application/vnd.google-earth.kmz"
	default:
		http.Error(w, `format must be "kmz" or "kml"`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func parseBBox(s string) (poi.BBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return poi.BBox{}, errBadBBox
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return poi.BBox{}, errBadBBox
		}
		vals[i] = f
	}
	return poi.BBox{MinLat: vals[0], MinLng: vals[1], MaxLat: vals[2], MaxLng: vals[3]}, nil
}

var errBadBBox = &httpError{"bbox must be minLat,minLng,maxLat,maxLng"}

type httpError struct{ s string }

func (e *httpError) Error() string { return e.s }

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) >= 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return h
}

func matchesAnyType(p poi.POI, types map[poi.Type]bool) bool {
	for t := range types {
		if p.MatchesType(t) {
			return true
		}
	}
	return false
}

func parseTypes(s string) map[poi.Type]bool {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	out := map[poi.Type]bool{}
	for _, p := range strings.Split(s, ",") {
		if t, ok := poi.ParseType(strings.TrimSpace(p)); ok {
			out[t] = true
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := s.cfg.Server.CORSOrigins
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originAllowed(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else if strings.TrimSpace(allowed) == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
