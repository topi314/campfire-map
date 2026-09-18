package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/topi314/campfire-export/server/campfire"
	"github.com/topi314/campfire-export/server/kml"
	"github.com/topi314/campfire-export/server/poi"
	"github.com/topi314/campfire-export/server/s2cells"
	"github.com/topi314/campfire-export/server/wayfarer"
)

const (
	headerWayfarerSession = "X-Wayfarer-Session"
	headerWayfarerXSRF    = "X-Wayfarer-XSRF"
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
	wantPowerspot := types == nil || types[poi.TypePowerspot]
	wayCreds := wayfarerCreds(r)
	useWayfarer := wayCreds.OK()

	if wantPowerspot && (r.Header.Get(headerWayfarerSession) != "" || r.Header.Get(headerWayfarerXSRF) != "") && !useWayfarer {
		http.Error(w, "Wayfarer requires both X-Wayfarer-Session and X-Wayfarer-XSRF", http.StatusBadRequest)
		return
	}

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
	dropTypes := campfire.DropTypesForToken(tok)
	if useWayfarer {
		dropTypes = campfire.DropTypesWithoutPowerspot()
	}
	key := strings.Join(cellIDs, ",") + ":" + campfire.CacheKeySuffix(tok)
	if useWayfarer {
		key += ":nowayspot"
	}

	needWayfarer := useWayfarer && wantPowerspot
	var pois, spots []poi.POI
	var campErr, wayErr error

	// Fetch Campfire GraphQL and Wayfarer mapview concurrently when both are needed.
	// Same cache keys coalesce across concurrent users (singleflight inside the cache).
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pois, campErr = s.loadCampfirePOIs(r, key, tok, cellIDs, level, dropTypes)
	}()
	if needWayfarer {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spots, wayErr = s.fetchWayfarerPowerspots(r, bbox, wayCreds)
		}()
	}
	wg.Wait()

	if campErr != nil {
		slog.Error("fetch pois", slog.Any("err", campErr))
		http.Error(w, "failed to fetch map data: "+campErr.Error(), http.StatusBadGateway)
		return
	}
	if wayErr != nil {
		var authErr *wayfarer.AuthError
		if errors.As(wayErr, &authErr) {
			http.Error(w, "Wayfarer session rejected. Paste fresh SESSION and XSRF-TOKEN.", http.StatusUnauthorized)
			return
		}
		slog.Error("fetch wayfarer powerspots", slog.Any("err", wayErr))
		http.Error(w, "failed to fetch Wayfarer powerspots: "+wayErr.Error(), http.StatusBadGateway)
		return
	}

	if useWayfarer {
		pois = stripPowerspots(pois)
		if wantPowerspot {
			pois = append(pois, spots...)
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

func (s *Server) loadCampfirePOIs(r *http.Request, sharedKey, tok string, cellIDs []string, level int, dropTypes []string) ([]poi.POI, error) {
	if raw, ok := s.cache.Get(sharedKey); ok {
		var pois []poi.POI
		if err := json.Unmarshal(raw, &pois); err == nil {
			return pois, nil
		}
	}

	// Coalesce duplicate in-flight requests per credential so one bad token
	// cannot fail other users; successful tiles are published to sharedKey.
	flightKey := sharedKey
	if tok != "" {
		flightKey = sharedKey + ":tok:" + shortHash(tok)
	}
	raw, err := s.cache.GetOrLoad(flightKey, func() ([]byte, error) {
		if raw, ok := s.cache.Get(sharedKey); ok {
			return raw, nil
		}
		pois, err := s.client.FetchWithDropTypes(campfire.WithToken(r.Context(), tok), cellIDs, level, dropTypes)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(pois)
		if err != nil {
			return nil, err
		}
		s.cache.Set(sharedKey, raw)
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	var pois []poi.POI
	if err := json.Unmarshal(raw, &pois); err != nil {
		return nil, err
	}
	return pois, nil
}

func (s *Server) fetchWayfarerPowerspots(r *http.Request, bbox poi.BBox, creds wayfarer.Creds) ([]poi.POI, error) {
	sharedKey := fmt.Sprintf("wayfarer:%g,%g,%g,%g", bbox.MinLat, bbox.MinLng, bbox.MaxLat, bbox.MaxLng)
	if raw, ok := s.cache.Get(sharedKey); ok {
		var pois []poi.POI
		if err := json.Unmarshal(raw, &pois); err == nil {
			return pois, nil
		}
	}

	flightKey := sharedKey + ":cred:" + shortHash(creds.Session+"\n"+creds.XSRF)
	raw, err := s.cache.GetOrLoad(flightKey, func() ([]byte, error) {
		if raw, ok := s.cache.Get(sharedKey); ok {
			return raw, nil
		}
		pois, err := s.wayfarer.FetchPowerspots(r.Context(), bbox, creds)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(pois)
		if err != nil {
			return nil, err
		}
		s.cache.Set(sharedKey, raw)
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	var pois []poi.POI
	if err := json.Unmarshal(raw, &pois); err != nil {
		return nil, err
	}
	return pois, nil
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func stripPowerspots(pois []poi.POI) []poi.POI {
	out := make([]poi.POI, 0, len(pois))
	for _, p := range pois {
		if p.Type == poi.TypePowerspot || p.Type == "dynaspot" || p.Type == "dmax" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func wayfarerCreds(r *http.Request) wayfarer.Creds {
	return wayfarer.Creds{
		Session: strings.TrimSpace(r.Header.Get(headerWayfarerSession)),
		XSRF:    strings.TrimSpace(r.Header.Get(headerWayfarerXSRF)),
	}
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Wayfarer-Session,X-Wayfarer-XSRF")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
