import type { Poi, PoiType } from "~/types/poi";

/** Büsingpark, Offenbach am Main */
export const DEFAULT_MAP_CENTER: [number, number] = [50.10056, 8.76167];
export const DEFAULT_MAP_ZOOM = 16;
export const DEFAULT_MAP_LABEL = "Büsingpark, Offenbach am Main";

/** Wayfarer-style minimum separation for planned campsite POIs (meters). */
export const WAYFARER_MIN_SEPARATION_M = 30;

/** Soft reminder: Community Ambassadors can typically submit this many POIs (not enforced). */
export const CAMPSITE_CA_SOFT_LIMIT = 25;

/** Approximate lure-module spawn range around a PokéStop (meters). */
export const LURE_RANGE_M = 40;

/** Current PokéStop / Gym interaction radius (meters). */
export const INTERACTION_RANGE_M = 80;

export type RadiusOverlayId = "wayfarer" | "lure" | "interaction";

export interface RadiusOverlayDef {
  id: RadiusOverlayId;
  label: string;
  meters: number;
  color: string;
  /** null = every non-route point POI */
  types: PoiType[] | null;
}

/** Campsite / export planning — Wayfarer-style spacing. */
export const EXPORT_RADIUS_OVERLAYS: RadiusOverlayDef[] = [
  {
    id: "wayfarer",
    label: "Min POI spacing",
    meters: WAYFARER_MIN_SEPARATION_M,
    color: "#e74c3c",
    types: null,
  },
];

/** General map overlays (Explore). */
export const MAP_RADIUS_OVERLAYS: RadiusOverlayDef[] = [
  {
    id: "lure",
    label: "Lure",
    meters: LURE_RANGE_M,
    color: "#e91e8c",
    types: ["pokestop"],
  },
  {
    id: "interaction",
    label: "Interaction",
    meters: INTERACTION_RANGE_M,
    color: "#3da0ff",
    types: ["gym", "super_mega_gym", "pokestop", "powerspot"],
  },
];

export const RADIUS_OVERLAYS: RadiusOverlayDef[] = [...EXPORT_RADIUS_OVERLAYS, ...MAP_RADIUS_OVERLAYS];

export type RadiusOverlayState = Record<RadiusOverlayId, boolean>;

export function defaultRadiusOverlays(): RadiusOverlayState {
  return { wayfarer: false, lure: false, interaction: false };
}

export function sanitizeRadiusOverlays(raw: unknown): RadiusOverlayState {
  const out = defaultRadiusOverlays();
  if (!raw || typeof raw !== "object") return out;
  const obj = raw as Record<string, unknown>;
  for (const def of RADIUS_OVERLAYS) {
    if (typeof obj[def.id] === "boolean") out[def.id] = obj[def.id];
  }
  return out;
}

export function radiusOverlayApplies(def: RadiusOverlayDef, p: Poi): boolean {
  if (p.type === "route") return false;
  if (!def.types) return true;
  return def.types.includes(p.type);
}

export function anyRadiusOverlayEnabled(state: RadiusOverlayState): boolean {
  return RADIUS_OVERLAYS.some((d) => state[d.id]);
}
