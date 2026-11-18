import type { Poi, PoiType } from "~/types/poi";

/** Büsingpark, Offenbach am Main */
export const DEFAULT_MAP_CENTER: [number, number] = [50.10056, 8.76167];
export const DEFAULT_MAP_ZOOM = 16;
export const DEFAULT_MAP_LABEL = "Büsingpark, Offenbach am Main";

/** Minimum separation for campsite POIs (meters). */
export const WAYFARER_MIN_SEPARATION_M = 30;

/** Soft reminder: Community Ambassadors can typically submit this many POIs (not enforced). */
export const CAMPSITE_CA_SOFT_LIMIT = 25;

/** Approximate lure-module spawn range around a PokéStop (meters). */
export const LURE_RANGE_M = 40;

/** Minimum spacing used for events (meters). */
export const POI_RANGE_M = 50;

/** Current PokéStop / Gym interaction radius (meters). */
export const INTERACTION_RANGE_M = 80;

export type RadiusOverlayId = "wayfarer" | "lure" | "poi" | "interaction";

export interface RadiusOverlayDef {
  id: RadiusOverlayId;
  label: string;
  meters: number;
  color: string;
  /** null = every non-route point POI */
  types: PoiType[] | null;
  /** Minimum-spacing rule. The too-close check uses whichever of these are enabled. */
  spacing?: boolean;
}

/** Map range overlays (Explore → Ranges; optional My Maps export layers). */
export const RADIUS_OVERLAYS: RadiusOverlayDef[] = [
  {
    id: "wayfarer",
    label: "Min spacing campsite",
    meters: WAYFARER_MIN_SEPARATION_M,
    color: "#e74c3c",
    types: null,
    spacing: true,
  },
  {
    id: "lure",
    label: "Lure",
    meters: LURE_RANGE_M,
    color: "#e91e8c",
    types: ["pokestop"],
  },
  {
    id: "poi",
    label: "Min spacing events",
    meters: POI_RANGE_M,
    color: "#2ecc71",
    types: null,
    spacing: true,
  },
  {
    id: "interaction",
    label: "Interaction",
    meters: INTERACTION_RANGE_M,
    color: "#3da0ff",
    types: ["gym", "super_mega_gym", "pokestop", "powerspot"],
  },
];

export type RadiusOverlayState = Record<RadiusOverlayId, boolean>;

export function defaultRadiusOverlays(): RadiusOverlayState {
  return { wayfarer: false, lure: false, poi: false, interaction: false };
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

/** Meters for the too-close check. Uses the strictest enabled min-spacing ring, or campsite spacing when none are on. */
export function activeMinSpacingMeters(state: RadiusOverlayState): number {
  let meters = 0;
  for (const def of RADIUS_OVERLAYS) {
    if (!def.spacing || !state[def.id]) continue;
    if (def.meters > meters) meters = def.meters;
  }
  return meters || WAYFARER_MIN_SEPARATION_M;
}

export function radiusOverlayApplies(def: RadiusOverlayDef, p: Poi): boolean {
  if (p.type === "route") return false;
  if (!def.types) return true;
  return def.types.includes(p.type);
}

export function anyRadiusOverlayEnabled(state: RadiusOverlayState): boolean {
  return RADIUS_OVERLAYS.some((d) => state[d.id]);
}
