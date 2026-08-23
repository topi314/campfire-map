import { BASE_MAPS, DEFAULT_BASE_MAP_ID } from "~/constants/baseMaps";
import { LAYER_TYPES, type PoiType } from "~/types/poi";

const STORAGE_KEY = "campfire-export.uiSettings";

export interface StoredUiSettings {
  enabled: Record<PoiType, boolean>;
  baseLayerId: string;
  showCells: boolean;
  exportMode: boolean;
  showAllRoutes: boolean;
  groupByLayer: boolean;
}

const DEFAULT_ENABLED: Record<PoiType, boolean> = {
  gym: true,
  super_mega_gym: false,
  pokestop: true,
  powerspot: false,
  route: true,
};

export function defaultUiSettings(): StoredUiSettings {
  return {
    enabled: { ...DEFAULT_ENABLED },
    baseLayerId: DEFAULT_BASE_MAP_ID,
    showCells: false,
    exportMode: false,
    showAllRoutes: false,
    groupByLayer: false,
  };
}

function sanitizeEnabled(raw: unknown): Record<PoiType, boolean> {
  const out = { ...DEFAULT_ENABLED };
  if (!raw || typeof raw !== "object") return out;
  const obj = raw as Record<string, unknown>;
  for (const t of LAYER_TYPES) {
    if (typeof obj[t] === "boolean") out[t] = obj[t];
  }
  return out;
}

function knownBaseLayerId(id: string) {
  return BASE_MAPS.some((m) => m.id === id);
}

function sanitizeSettings(raw: unknown): StoredUiSettings {
  const defaults = defaultUiSettings();
  if (!raw || typeof raw !== "object") return defaults;
  const obj = raw as Record<string, unknown>;
  const baseLayerId = typeof obj.baseLayerId === "string" ? obj.baseLayerId : defaults.baseLayerId;
  const enabled = sanitizeEnabled(obj.enabled);
  const showAllRoutes =
    typeof obj.showAllRoutes === "boolean" ? obj.showAllRoutes : defaults.showAllRoutes;
  const groupByLayer =
    typeof obj.groupByLayer === "boolean"
      ? obj.groupByLayer
      : typeof obj.sortByLayer === "boolean"
        ? obj.sortByLayer
        : defaults.groupByLayer;
  return {
    enabled,
    baseLayerId: knownBaseLayerId(baseLayerId) ? baseLayerId : defaults.baseLayerId,
    showCells: typeof obj.showCells === "boolean" ? obj.showCells : defaults.showCells,
    exportMode: typeof obj.exportMode === "boolean" ? obj.exportMode : defaults.exportMode,
    showAllRoutes: enabled.route ? showAllRoutes : false,
    groupByLayer,
  };
}

export function readStoredUiSettings(): StoredUiSettings {
  if (!import.meta.client) return defaultUiSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultUiSettings();
    return sanitizeSettings(JSON.parse(raw));
  } catch {
    return defaultUiSettings();
  }
}

export function writeStoredUiSettings(settings: StoredUiSettings) {
  if (!import.meta.client) return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sanitizeSettings(settings)));
  } catch {
    /* private mode */
  }
}
