import {
  LAYER_TYPES,
  TYPE_META,
  isGymPoi,
  isInactivePowerspot,
  isSuperMegaPoi,
  powerspotDisplayName,
  type Poi,
  type PoiType,
} from "~/types/poi";

export interface ExportSettings {
  mapName: string;
  fileName: string;
  format: "kmz" | "kml";
  includeTypes: Record<PoiType, boolean>;
  includeRoutePaths: boolean;
  includeInactivePowerspots: boolean;
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

export function filterPoisForExport(pois: Poi[], settings: ExportSettings): Poi[] {
  const out: Poi[] = [];
  for (const p of pois) {
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
    if (p.type === "powerspot" && isInactivePowerspot(p)) {
      out.push({ ...p, name: powerspotDisplayName(p) });
      continue;
    }
    out.push(p);
  }
  return out;
}

export function exportTypeLabels(): { type: PoiType; label: string }[] {
  return LAYER_TYPES.map((type) => ({
    type,
    label: TYPE_META[type].label,
  }));
}
