# Pokémon GO map

Go proxy + Nuxt/Leaflet map viewer for Pokémon GO POIs. Pan the map to load gyms, stops, powerspots, and routes for the current view; filter layers; optionally export a Google My Maps–ready KMZ/KML.

The browser never talks to Niantic. The backend fetches S2 tiles from `https://niantic-social-api.nianticlabs.com/graphql`, caches them for 24 hours, and returns a normalized POI list.

In production the Nuxt SPA is built into the Go binary (`embed`) and served from the same process — one Docker image.

## POI layers

Gyms, Super Mega Gyms (`isMegaEnhancedEligible` / mega-enhanced raid), PokéStops, Powerspots, and Routes. Duplicate “pokestops” in the original request is treated as PokéStops. Route waypoints that are already stops are not duplicated.

Each feature has **id, type, name, icon URL, lat, lng**. Routes also include a path.

## Run locally

### Backend (+ embedded UI placeholder)

```bash
cp example.config.toml config.toml
go test ./...
go run . -config config.toml
```

Listens on `:8080` (API + placeholder UI until you generate the frontend into `server/web/dist`).

### Frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:3000. In dev, `/api` is proxied to the Go server on `:8080`.

### Embed the SPA into Go (optional local prod-like)

```bash
cd frontend
NUXT_PUBLIC_API_BASE= npm run generate
# PowerShell:
Remove-Item -Recurse -Force ..\server\web\dist\*
Copy-Item -Recurse .output\public\* ..\server\web\dist\
cd ..
go run . -config config.toml
```

Then open http://localhost:8080 — API and UI on the same origin.

### Docker (single image)

```bash
cp example.config.toml config.toml
docker compose up --build
```

Open http://localhost:8080.

## Export to Google My Maps

Optional workflow when you want a My Maps file:

1. Open **Export**, then draw a rectangle or click markers (or **Select visible**).
2. Download **KMZ** or **KML**.
3. In [Google My Maps](https://www.google.com/maps/d/) create a map and **Import** the file.

My Maps allows 10 layers and 2000 features per layer. The export uses one folder per type and splits overflow (`Gyms 2`, …). There is no public API to create a My Map.

## API

- `GET /api/health`
- `GET /api/pois?bbox=minLat,minLng,maxLat,maxLng&types=gym,super_mega_gym,pokestop,powerspot,route`
- `POST /api/export` `{ "name": "…", "format": "kmz"|"kml", "pois": [ … ] }` → KMZ or KML download

World dumps are rejected (`limits.max_bbox_span`, default 0.35°).

GraphQL operations live in [`server/campfire/queries`](server/campfire/queries/README.md).
