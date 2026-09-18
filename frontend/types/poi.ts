export type PoiType = "gym" | "super_mega_gym" | "pokestop" | "powerspot" | "route";

export type PowerspotStatus = "active" | "inactive";

export interface MapCell {
  id: string;
  level: number;
  ring: [number, number][];
}

export interface Poi {
  id: string;
  type: PoiType;
  name: string;
  icon: string;
  lat: number;
  lng: number;
  path?: [number, number][];
  superMegaEligible?: boolean;
  /** Set for Wayfarer powerspots. */
  status?: PowerspotStatus | string;
}

export const TYPE_META: Record<
  PoiType,
  { label: string; color: string; short: string; image: string }
> = {
  gym: { label: "Gyms", color: "#a4b0bf", short: "G", image: "/images/pgo-gym.svg" },
  super_mega_gym: {
    label: "Super Mega Gyms",
    color: "#9b59b6",
    short: "SM",
    image: "/images/super-mega-gym.svg",
  },
  pokestop: { label: "PokéStops", color: "#7fcafe", short: "S", image: "/images/pgo-pokestop.svg" },
  powerspot: { label: "Powerspot", color: "#f481c4", short: "P", image: "/images/pgo-powerspot.svg" },
  route: { label: "Routes", color: "#3da0ff", short: "R", image: "/images/pgo-route.svg" },
};

/** Darker powerspot styling for inactive Wayfarer spots (same glyph, darker tint). */
export const INACTIVE_POWERSPOT = {
  label: "Inactive",
  color: "#6b2d4a",
  image: "/images/pgo-powerspot.svg",
};

/** Route line and endpoint colors (official map). */
export const ROUTE_COLORS = {
  line: "#3da0ff",
  start: "#01a3ee",
  end: "#ff4747",
};

export const ALL_TYPES = Object.keys(TYPE_META) as PoiType[];

export const LAYER_TYPES: PoiType[] = ["gym", "super_mega_gym", "pokestop", "powerspot", "route"];

export function normalizePoiType(t: string): PoiType {
  if (t === "dmax" || t === "dynaspot") return "powerspot";
  return t as PoiType;
}

export function normalizePoi(p: Poi): Poi {
  const type = normalizePoiType(p.type);
  const status = typeof p.status === "string" ? p.status.toLowerCase() : p.status;
  return type === p.type && status === p.status ? p : { ...p, type, status };
}

export function isGymPoi(p: Poi) {
  return p.type === "gym" || p.type === "super_mega_gym" || !!p.superMegaEligible;
}

export function isSuperMegaPoi(p: Poi) {
  return !!p.superMegaEligible || p.type === "super_mega_gym";
}

export function isInactivePowerspot(p: Poi) {
  return normalizePoiType(p.type) === "powerspot" && p.status === "inactive";
}

export function poiDisplayType(p: Poi, superMegaLayerOn: boolean): PoiType {
  if (isGymPoi(p) && isSuperMegaPoi(p) && superMegaLayerOn) {
    return "super_mega_gym";
  }
  if (isGymPoi(p)) {
    return "gym";
  }
  return normalizePoiType(p.type);
}

export function poiLayerVisible(
  p: Poi,
  enabled: Record<PoiType, boolean>,
  opts?: { showInactivePowerspots?: boolean },
) {
  if (isGymPoi(p)) {
    return !!enabled.gym || (!!enabled.super_mega_gym && isSuperMegaPoi(p));
  }
  const type = normalizePoiType(p.type);
  if (!enabled[type]) return false;
  if (type === "powerspot" && isInactivePowerspot(p) && opts?.showInactivePowerspots === false) {
    return false;
  }
  return true;
}
