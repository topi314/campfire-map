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

### CARTO basemap key

CARTO raster basemaps (Voyager, Positron, Dark matter) show an “API key required” watermark without a key. Request a free one at [carto.com/basemaps/apikey](https://carto.com/basemaps/apikey) and put it in `config.toml`:

```toml
[basemaps]
carto_api_key = "your_carto_key"
```

The browser reads it from `GET /api/config` and appends it to CARTO tile URLs. Other basemaps ignore it.

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

1. Open **Export**, then select POIs (click markers, or **Draw area** to select everything inside a polygon).
2. Use **Lock map** to freeze fetches and show only the selection. Place planned PokéStops / Gyms / Powerspots (too-close spots are marked in red).
3. In **Export…**, assign POI types to named reorderable layers (**Campsite layout** or **By type**). Routes keep start, path, and end in one layer.
4. Download **KMZ** or **KML**, or **Save work** to reopen later (**Open work**).
5. In [Google My Maps](https://www.google.com/maps/d/) create a map and **Import** the KMZ/KML. My Maps will show individual styles per place — that’s expected.

My Maps allows 10 layers and 2000 features per layer. Overflow folders are named `Existing 2`, … There is no public API to create a My Map.

On desktop, Explore (and Export when open) stay as side columns. On mobile, controls use a draggable bottom sheet (peek / half / expanded).

## API

- `GET /api/health`
- `GET /api/config` → `{ "cartoApiKey": "…" }` (basemap key for the browser, from `[basemaps]`)
- `GET /api/pois?bbox=minLat,minLng,maxLat,maxLng&types=gym,super_mega_gym,pokestop,powerspot,route`
- `POST /api/export` `{ "name": "…", "format": "kmz"|"kml", "pois": [ … ], "outline"?: [[lat,lng],…], "layers"?: [{ "name", "types": [...] }] }` → KMZ or KML download

World dumps are rejected (`limits.max_bbox_span`, default 0.35°).

GraphQL operations live in [`server/campfire/queries`](server/campfire/queries/README.md).
