# Niantic GraphQL map operations

Captured from [pokemongo.com/map](https://pokemongo.com/map) → Niantic social GraphQL.

| File                              | Operation                                                      | When              |
|-----------------------------------|----------------------------------------------------------------|-------------------|
| `map_objects_by_s2_cells.graphql` | `PgoGameMapObjectsByS2CellsProvider_mapObjectsByS2Cells_Query` | Viewport pan/zoom |

This is the only Campfire query the backend uses. Names, images, and route paths are requested inline (no preview-card enrich pass).

## Variables

Built in Go from viewport S2 cells:

- `realityChannelId` — `campfire.reality_channel_id`
- `s2CellLevel` — always **15** (full POI details)
- Overlay (S2 cells toggle) may still draw L14/L17 cells for visualization
- `sourcesByS2Cells` — per cell: `{ s2CellId, sources: [{ name: "PGO", dropTypes }] }`

Drop types: `PGO_GYM`, `PGO_POWERSPOT`, `PGO_POKESTOP`, `PGO_ROUTE`.

## Auth

Browser session token → `Authorization: Bearer` (not in server config).

On Campfire web, the value lives in Local Storage as `CapacitorStorage.sessionToken`.
Copy it from DevTools → Application/Storage → Local Storage (Chrome, Firefox, or Safari).
