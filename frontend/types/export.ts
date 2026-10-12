import {
  RADIUS_OVERLAYS,
  radiusOverlayApplies,
  type RadiusOverlayDef,
  type RadiusOverlayId,
} from "~/constants/map";
import {
  LAYER_TYPES,
  TYPE_META,
  isGymPoi,
  isInactivePowerspot,
  isSuperMegaPoi,
  type Poi,
  type PoiType,
} from "~/types/poi";

export type PlaceTool = "pokestop" | "gym" | "powerspot" | "outline" | "draw_route" | null;

/** Assignable My Maps layer type chips (not individual POIs). */
export type LayerTypeKey =
  | PoiType
  | "campsite_gym"
  | "campsite_pokestop"
  | "campsite_powerspot"
  | "outline"
  | "drawn_route"
  | "range_wayfarer"
  | "range_lure"
  | "range_interaction";

export interface ExportLayer {
  id: string;
  name: string;
  types: LayerTypeKey[];
}

export interface ExportSettings {
  mapName: string;
  fileName: string;
  format: "kmz" | "kml";
  includeTypes: Record<PoiType, boolean>;
  includeRoutePaths: boolean;
  includeInactivePowerspots: boolean;
  layers: ExportLayer[];
}

export const EXISTING_LAYER_TYPES: LayerTypeKey[] = [
  "gym",
  "super_mega_gym",
  "pokestop",
  "powerspot",
  "route",
];

export const CAMPSITE_LAYER_TYPES: LayerTypeKey[] = [
  "campsite_gym",
  "campsite_pokestop",
  "campsite_powerspot",
  "outline",
  "drawn_route",
];

export const RANGE_LAYER_TYPES: LayerTypeKey[] = [
  "range_wayfarer",
  "range_lure",
  "range_interaction",
];

export const ALL_LAYER_TYPE_KEYS: LayerTypeKey[] = [
  ...EXISTING_LAYER_TYPES,
  ...CAMPSITE_LAYER_TYPES,
  ...RANGE_LAYER_TYPES,
];

export const LAYER_TYPE_LABELS: Record<LayerTypeKey, string> = {
  gym: "Gyms",
  super_mega_gym: "Super Mega Gyms",
  pokestop: "PokéStops",
  powerspot: "Powerspots",
  route: "Routes",
  campsite_gym: "Campsite Gyms",
  campsite_pokestop: "Campsite PokéStops",
  campsite_powerspot: "Campsite Powerspots",
  outline: "Areas",
  drawn_route: "Campsite Routes",
  range_wayfarer: "Range · Min POI spacing (30 m)",
  range_lure: "Range · Lure (40 m)",
  range_interaction: "Range · Interaction (80 m)",
};

export function isRangeLayerType(t: LayerTypeKey): boolean {
  return RANGE_LAYER_TYPES.includes(t);
}

export function rangeOverlayIdForLayerType(t: LayerTypeKey): RadiusOverlayId | null {
  switch (t) {
    case "range_wayfarer":
      return "wayfarer";
    case "range_lure":
      return "lure";
    case "range_interaction":
      return "interaction";
    default:
      return null;
  }
}

export function rangeDefForLayerType(t: LayerTypeKey): RadiusOverlayDef | null {
  const id = rangeOverlayIdForLayerType(t);
  if (!id) return null;
  return RADIUS_OVERLAYS.find((d) => d.id === id) ?? null;
}

/** Campsite planner: existing neighbors (incl. routes), areas, planned campsite pins. */
export function defaultCampsiteExportLayers(): ExportLayer[] {
  return [
    {
      id: "layer-existing",
      name: "Existing",
      types: ["gym", "pokestop", "powerspot", "route"],
    },
    {
      id: "layer-play-area",
      name: "Areas",
      types: ["outline"],
    },
    {
      id: "layer-guide-routes",
      name: "Campsite Routes",
      types: ["drawn_route"],
    },
    {
      id: "layer-ranges",
      name: "Ranges",
      types: ["range_wayfarer"],
    },
    {
      id: "layer-campsite",
      name: "Campsite",
      types: ["campsite_gym", "campsite_pokestop", "campsite_powerspot"],
    },
  ];
}

/** General website / explore export: one My Maps layer per POI type. */
export function defaultWebsiteExportLayers(): ExportLayer[] {
  return [
    { id: "layer-pokestop", name: "PokéStops", types: ["pokestop"] },
    { id: "layer-gym", name: "Gyms", types: ["gym"] },
    { id: "layer-super-mega-gym", name: "Super Mega Gyms", types: ["super_mega_gym"] },
    { id: "layer-powerspot", name: "Powerspots", types: ["powerspot"] },
    { id: "layer-route", name: "Routes", types: ["route"] },
  ];
}

/** Default for new installs / unset settings (campsite planner). */
export function defaultExportLayers(): ExportLayer[] {
  return defaultCampsiteExportLayers();
}

export const DEFAULT_EXPORT_SETTINGS: ExportSettings = {
  mapName: "Pokémon GO map",
  fileName: "pogo-map",
  format: "kmz",
  includeTypes: {
    gym: true,
    super_mega_gym: true,
    pokestop: true,
    powerspot: true,
    route: true,
  },
  includeRoutePaths: true,
  includeInactivePowerspots: true,
  layers: defaultExportLayers(),
};

export function sanitizeFileName(name: string) {
  const base = name
    .trim()
    .replace(/[<>:"/\\|?*]+/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
  return base || "pogo-map";
}

export function newLayerId() {
  return `layer-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

/** Normalize saved layers: unique type assignment; auto-place missing campsite_* on a Campsite layer. */
export function normalizeExportLayers(layers: ExportLayer[] | undefined | null): ExportLayer[] {
  const base = layers?.length ? layers.map((l) => ({ ...l, types: [...l.types] })) : defaultExportLayers();
  const seen = new Set<LayerTypeKey>();
  for (const layer of base) {
    layer.types = layer.types.filter((t) => {
      if (!ALL_LAYER_TYPE_KEYS.includes(t) || seen.has(t)) return false;
      seen.add(t);
      return true;
    });
  }
  // Older saved settings predate campsite_* chips — without this, planned POIs are omitted.
  const campsiteKeys: LayerTypeKey[] = ["campsite_gym", "campsite_pokestop", "campsite_powerspot"];
  const missing = campsiteKeys.filter((k) => !seen.has(k));
  if (missing.length) {
    let camp = base.find((l) => /campsite/i.test(l.name));
    if (!camp) {
      camp = { id: newLayerId(), name: "Campsite", types: [] };
      base.push(camp);
    }
    camp.types.push(...missing);
  }
  // Fold a dedicated Routes-only layer into Existing (campsite default no longer has one).
  const routesOnlyIdx = base.findIndex(
    (l) => /^routes$/i.test(l.name.trim()) && l.types.length === 1 && l.types[0] === "route",
  );
  if (routesOnlyIdx >= 0) {
    const existing = base.find((l) => /existing/i.test(l.name));
    if (existing && !existing.types.includes("route")) {
      existing.types.push("route");
      seen.add("route");
    }
    if (existing) base.splice(routesOnlyIdx, 1);
  }
  // Routes stay with Existing when unassigned.
  if (!seen.has("route")) {
    const existing = base.find((l) => /existing/i.test(l.name));
    if (existing) {
      existing.types.push("route");
      seen.add("route");
    }
  }
  // Outline layer was formerly labeled "Play Area".
  for (const layer of base) {
    if (layer.name === "Play Area" && layer.types.includes("outline")) {
      layer.name = "Areas";
    }
    if (
      (layer.name === "Guide routes" || layer.name === "Routes") &&
      layer.types.includes("drawn_route")
    ) {
      layer.name = "Campsite Routes";
    }
  }
  // Drawn routes layer (separate from Campfire / PoGo routes).
  if (!seen.has("drawn_route")) {
    const afterAreas = base.findIndex((l) => l.types.includes("outline") || /^areas$/i.test(l.name));
    const routesLayer: ExportLayer = {
      id: newLayerId(),
      name: "Campsite Routes",
      types: ["drawn_route"],
    };
    if (afterAreas >= 0) base.splice(afterAreas + 1, 0, routesLayer);
    else base.push(routesLayer);
    seen.add("drawn_route");
  }
  return base;
}

/** Layer chips shown for the current Include types checkboxes. */
export function availableLayerTypeKeys(includeTypes: Record<PoiType, boolean>): LayerTypeKey[] {
  return ALL_LAYER_TYPE_KEYS.filter((t) => campsiteLayerTypeIncluded(t, includeTypes));
}

export function unassignedLayerTypes(
  layers: ExportLayer[],
  includeTypes?: Record<PoiType, boolean>,
): LayerTypeKey[] {
  const pool = includeTypes ? availableLayerTypeKeys(includeTypes) : ALL_LAYER_TYPE_KEYS;
  const assigned = new Set(layers.flatMap((l) => l.types));
  return pool.filter((t) => !assigned.has(t));
}

export function layerTypeKeyForPoi(p: Poi): LayerTypeKey {
  if (p.source === "campsite") {
    if (p.type === "gym" || p.type === "super_mega_gym") return "campsite_gym";
    if (p.type === "pokestop") return "campsite_pokestop";
    if (p.type === "powerspot") return "campsite_powerspot";
  }
  return p.type;
}

export function filterPoisForExport(pois: Poi[], settings: ExportSettings): Poi[] {
  const out: Poi[] = [];
  for (const p of pois) {
    if (p.source === "campsite") {
      const baseType: PoiType =
        p.type === "super_mega_gym" ? "gym" : (p.type as PoiType);
      if (baseType === "gym" && !settings.includeTypes.gym) continue;
      if (baseType === "pokestop" && !settings.includeTypes.pokestop) continue;
      if (baseType === "powerspot" && !settings.includeTypes.powerspot) continue;
      out.push({
        ...p,
        type: baseType === "gym" ? "gym" : baseType,
        source: "campsite",
      });
      continue;
    }
    if (isGymPoi(p)) {
      if (settings.includeTypes.gym) {
        out.push({ ...p, type: "gym" });
      }
      if (settings.includeTypes.super_mega_gym && isSuperMegaPoi(p)) {
        out.push({ ...p, type: "super_mega_gym" });
      }
      continue;
    }
    if (!settings.includeTypes[p.type]) continue;
    if (p.type === "powerspot" && isInactivePowerspot(p) && !settings.includeInactivePowerspots) {
      continue;
    }
    if (p.type === "route" && !settings.includeRoutePaths) {
      const { path: _path, ...rest } = p;
      out.push(rest);
      continue;
    }
    out.push(p);
  }
  return out;
}

/** Estimate My Maps folder count from user layers + 2000-feature splits. */
export function estimateMyMapsFolders(
  pois: Poi[],
  layers: ExportLayer[],
  areaCount: number,
  drawnRouteCount = 0,
): number {
  const counts = new Map<string, number>();
  for (const layer of layers) {
    counts.set(layer.id, 0);
  }
  const typeToLayer = new Map<LayerTypeKey, string>();
  for (const layer of layers) {
    for (const t of layer.types) typeToLayer.set(t, layer.id);
  }
  for (const p of pois) {
    const key = layerTypeKeyForPoi(p);
    const layerId = typeToLayer.get(key);
    if (!layerId) continue;
    // Routes with path: start + path (+ end when open). Estimate uses 3 (upper bound).
    let n = 1;
    if (p.type === "route" && p.path && p.path.length > 1) n = 3;
    counts.set(layerId, (counts.get(layerId) || 0) + n);
  }
  if (areaCount > 0) {
    const layerId = typeToLayer.get("outline");
    if (layerId) counts.set(layerId, (counts.get(layerId) || 0) + areaCount);
  }
  if (drawnRouteCount > 0) {
    const layerId = typeToLayer.get("drawn_route");
    // Each drawn route: start + path (+ end when open). Estimate uses 3.
    if (layerId) counts.set(layerId, (counts.get(layerId) || 0) + drawnRouteCount * 3);
  }
  for (const t of RANGE_LAYER_TYPES) {
    const layerId = typeToLayer.get(t);
    const def = rangeDefForLayerType(t);
    if (!layerId || !def) continue;
    let n = 0;
    for (const p of pois) {
      if (radiusOverlayApplies(def, p)) n++;
    }
    if (n) counts.set(layerId, (counts.get(layerId) || 0) + n);
  }
  let folders = 0;
  for (const n of counts.values()) {
    if (n <= 0) continue;
    folders += Math.ceil(n / 2000);
  }
  return folders;
}

export function exportTypeLabels(): { type: PoiType; label: string }[] {
  return LAYER_TYPES.map((type) => ({
    type,
    label: TYPE_META[type].label,
  }));
}

export function campsiteLayerTypeIncluded(key: LayerTypeKey, includeTypes: Record<PoiType, boolean>) {
  switch (key) {
    case "campsite_gym":
      return includeTypes.gym;
    case "campsite_pokestop":
      return includeTypes.pokestop;
    case "campsite_powerspot":
      return includeTypes.powerspot;
    case "outline":
    case "drawn_route":
    case "range_wayfarer":
      return true;
    case "range_lure":
      return !!includeTypes.pokestop;
    case "range_interaction":
      return !!(
        includeTypes.gym ||
        includeTypes.super_mega_gym ||
        includeTypes.pokestop ||
        includeTypes.powerspot
      );
    default:
      return includeTypes[key as PoiType] ?? true;
  }
}
