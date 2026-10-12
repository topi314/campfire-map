import {
  DEFAULT_EXPORT_SETTINGS,
  normalizeExportLayers,
  type ExportLayer,
  type ExportSettings,
} from "~/types/export";
import { normalizePoi, type Poi } from "~/types/poi";

export const PLAN_FILE_VERSION = 3;
export const PLAN_FILE_SUFFIX = ".campfire-plan.json";

/** Default stroke/fill for map areas (matches prior single-outline blue). */
export const DEFAULT_AREA_COLOR = "#5b8cff";

/** Cycled when creating new areas/routes so zones are visually distinct. */
export const AREA_COLOR_PALETTE = [
  "#5b8cff",
  "#e85d4c",
  "#3dbb7a",
  "#f0a202",
  "#9b6bff",
  "#00b4d8",
  "#ef476f",
  "#06d6a0",
] as const;

export type MapArea = {
  id: string;
  name: string;
  color: string; // #rrggbb
  ring: [number, number][]; // lat,lng, ≥3
};

export type MapRoute = {
  id: string;
  name: string;
  color: string; // #rrggbb
  path: [number, number][]; // lat,lng, ≥2
};

export function normalizeAreaColor(raw: unknown): string {
  if (typeof raw !== "string") return DEFAULT_AREA_COLOR;
  const m = raw.trim().match(/^#?([0-9a-fA-F]{6})$/);
  if (!m) return DEFAULT_AREA_COLOR;
  return `#${m[1].toLowerCase()}`;
}

export function areaColorAt(index: number): string {
  return AREA_COLOR_PALETTE[index % AREA_COLOR_PALETTE.length];
}

/** Next palette color not in usedColors, searching from startIndex (wraps). */
export function nextPaletteColor(usedColors: string[], startIndex = 0): string {
  const used = new Set(usedColors.map(normalizeAreaColor));
  for (let i = 0; i < AREA_COLOR_PALETTE.length; i++) {
    const c = areaColorAt(startIndex + i);
    if (!used.has(c)) return c;
  }
  return areaColorAt(startIndex + used.size);
}

export interface CampfirePlan {
  version: number;
  mapName: string;
  fileName: string;
  existingPois: Poi[];
  campsitePois: Poi[];
  areas: MapArea[];
  routes: MapRoute[];
  layers: ExportLayer[];
  exportSettings: Omit<ExportSettings, "layers" | "mapName" | "fileName">;
}

function newAreaId() {
  return `area-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function newRouteId() {
  return `route-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function parseLatLngList(raw: unknown, minLen: number, label: string): [number, number][] | null {
  if (!Array.isArray(raw)) return null;
  const pts = raw.map((pt) => {
    if (!Array.isArray(pt) || pt.length < 2) throw new Error(`Invalid ${label} point`);
    return [Number(pt[0]), Number(pt[1])] as [number, number];
  });
  return pts.length >= minLen ? pts : null;
}

function parseAreas(o: Record<string, unknown>): MapArea[] {
  if (Array.isArray(o.areas)) {
    const out: MapArea[] = [];
    for (const item of o.areas) {
      if (!item || typeof item !== "object") continue;
      const a = item as Record<string, unknown>;
      const ring = parseLatLngList(a.ring, 3, "area");
      if (!ring) continue;
      const id = typeof a.id === "string" && a.id.trim() ? a.id : newAreaId();
      const name =
        typeof a.name === "string" && a.name.trim() ? a.name.trim() : "Area";
      const color = normalizeAreaColor(a.color);
      out.push({ id, name, color, ring });
    }
    return out;
  }
  // v1: single outline → Play Area
  const ring = parseLatLngList(o.outline, 3, "area");
  if (!ring) return [];
  return [{ id: newAreaId(), name: "Play Area", color: DEFAULT_AREA_COLOR, ring }];
}

function parseRoutes(o: Record<string, unknown>): MapRoute[] {
  if (!Array.isArray(o.routes)) return [];
  const out: MapRoute[] = [];
  for (const item of o.routes) {
    if (!item || typeof item !== "object") continue;
    const r = item as Record<string, unknown>;
    const path = parseLatLngList(r.path, 2, "route");
    if (!path) continue;
    const id = typeof r.id === "string" && r.id.trim() ? r.id : newRouteId();
    const name =
      typeof r.name === "string" && r.name.trim() ? r.name.trim() : "Route";
    const color = normalizeAreaColor(r.color);
    out.push({ id, name, color, path });
  }
  return out;
}

export function buildCampfirePlan(input: {
  mapName: string;
  fileName: string;
  existingPois: Poi[];
  campsitePois: Poi[];
  areas: MapArea[];
  routes: MapRoute[];
  settings: ExportSettings;
}): CampfirePlan {
  const { layers, mapName: _m, fileName: _f, ...rest } = input.settings;
  return {
    version: PLAN_FILE_VERSION,
    mapName: input.mapName,
    fileName: input.fileName,
    existingPois: input.existingPois,
    campsitePois: input.campsitePois,
    areas: input.areas.filter((a) => a.ring.length >= 3),
    routes: input.routes.filter((r) => r.path.length >= 2),
    layers: normalizeExportLayers(layers),
    exportSettings: rest,
  };
}

export function parseCampfirePlan(raw: unknown): CampfirePlan {
  if (!raw || typeof raw !== "object") throw new Error("Invalid plan file");
  const o = raw as Record<string, unknown>;
  const version = o.version;
  if (version !== 1 && version !== 2 && version !== PLAN_FILE_VERSION) {
    throw new Error(`Unsupported plan version (expected 1, 2, or ${PLAN_FILE_VERSION})`);
  }
  if (!Array.isArray(o.existingPois) || !Array.isArray(o.campsitePois)) {
    throw new Error("Plan must include existingPois and campsitePois arrays");
  }
  const existingPois = (o.existingPois as Poi[]).map(normalizePoi);
  const campsitePois = (o.campsitePois as Poi[]).map((p) =>
    normalizePoi({ ...p, source: "campsite" }),
  );
  const areas = parseAreas(o);
  const routes = parseRoutes(o);
  const layers = normalizeExportLayers(o.layers as ExportLayer[] | undefined);
  const saved = (o.exportSettings || {}) as Partial<ExportSettings>;
  const exportSettings: CampfirePlan["exportSettings"] = {
    format: saved.format === "kml" ? "kml" : "kmz",
    includeTypes: {
      ...DEFAULT_EXPORT_SETTINGS.includeTypes,
      ...saved.includeTypes,
    },
    includeRoutePaths:
      typeof saved.includeRoutePaths === "boolean"
        ? saved.includeRoutePaths
        : DEFAULT_EXPORT_SETTINGS.includeRoutePaths,
    includeInactivePowerspots:
      typeof saved.includeInactivePowerspots === "boolean"
        ? saved.includeInactivePowerspots
        : DEFAULT_EXPORT_SETTINGS.includeInactivePowerspots,
  };
  return {
    version: PLAN_FILE_VERSION,
    mapName: typeof o.mapName === "string" && o.mapName.trim() ? o.mapName : DEFAULT_EXPORT_SETTINGS.mapName,
    fileName: typeof o.fileName === "string" && o.fileName.trim() ? o.fileName : DEFAULT_EXPORT_SETTINGS.fileName,
    existingPois,
    campsitePois,
    areas,
    routes,
    layers,
    exportSettings,
  };
}

export function downloadCampfirePlan(plan: CampfirePlan, fileName: string) {
  const blob = new Blob([JSON.stringify(plan, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName.endsWith(PLAN_FILE_SUFFIX) ? fileName : `${fileName}${PLAN_FILE_SUFFIX}`;
  a.click();
  URL.revokeObjectURL(url);
}

export async function readCampfirePlanFile(file: File): Promise<CampfirePlan> {
  const text = await file.text();
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch {
    throw new Error("File is not valid JSON");
  }
  return parseCampfirePlan(raw);
}
