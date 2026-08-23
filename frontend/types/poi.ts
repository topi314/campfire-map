export type PoiType = "gym" | "super_mega_gym" | "pokestop" | "powerspot" | "route";

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
  return type === p.type ? p : { ...p, type };
}

export function isGymPoi(p: Poi) {
  return p.type === "gym" || p.type === "super_mega_gym" || !!p.superMegaEligible;
}

export function isSuperMegaPoi(p: Poi) {
  return !!p.superMegaEligible || p.type === "super_mega_gym";
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

export function poiLayerVisible(p: Poi, enabled: Record<PoiType, boolean>) {
  if (isGymPoi(p)) {
    return !!enabled.gym || (!!enabled.super_mega_gym && isSuperMegaPoi(p));
  }
  return !!enabled[normalizePoiType(p.type)];
}
