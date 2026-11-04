import { BASE_MAPS, DEFAULT_BASE_MAP_ID } from "~/constants/baseMaps";
import {
  defaultRadiusOverlays,
  sanitizeRadiusOverlays,
  type RadiusOverlayState,
} from "~/constants/map";
import { LAYER_TYPES, type PoiType } from "~/types/poi";

const STORAGE_KEY = "campfire-export.uiSettings";

export interface StoredUiSettings {
  enabled: Record<PoiType, boolean>;
  baseLayerId: string;
  showCells: boolean;
  showL14: boolean;
  showL17: boolean;
  emptyCells: boolean;
  exportMode: boolean;
  showAllRoutes: boolean;
  showInactivePowerspots: boolean;
  groupByLayer: boolean;
  radiusOverlays: RadiusOverlayState;
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
    showL14: true,
    showL17: true,
    emptyCells: false,
    exportMode: false,
    showAllRoutes: false,
    showInactivePowerspots: true,
    groupByLayer: false,
    radiusOverlays: defaultRadiusOverlays(),
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
  const showInactivePowerspots =
    typeof obj.showInactivePowerspots === "boolean"
      ? obj.showInactivePowerspots
      : defaults.showInactivePowerspots;
  const groupByLayer =
    typeof obj.groupByLayer === "boolean"
      ? obj.groupByLayer
      : typeof obj.sortByLayer === "boolean"
        ? obj.sortByLayer
        : defaults.groupByLayer;
  const hasMaster = typeof obj.showCells === "boolean";
  const hasLevel = typeof obj.showL14 === "boolean" || typeof obj.showL17 === "boolean";
  const showL14 =
    typeof obj.showL14 === "boolean"
      ? obj.showL14
      : hasMaster
        ? (obj.showCells as boolean)
        : defaults.showL14;
  const showL17 =
    typeof obj.showL17 === "boolean"
      ? obj.showL17
      : hasMaster
        ? (obj.showCells as boolean)
        : defaults.showL17;
  const showCells = hasMaster && hasLevel
    ? (obj.showCells as boolean)
    : hasLevel
      ? showL14 || showL17
      : hasMaster
        ? (obj.showCells as boolean)
        : defaults.showCells;
  return {
    enabled,
    baseLayerId: knownBaseLayerId(baseLayerId) ? baseLayerId : defaults.baseLayerId,
    showCells,
    showL14,
    showL17,
    emptyCells: typeof obj.emptyCells === "boolean" ? obj.emptyCells : defaults.emptyCells,
    exportMode: typeof obj.exportMode === "boolean" ? obj.exportMode : defaults.exportMode,
    showAllRoutes: enabled.route ? showAllRoutes : false,
    showInactivePowerspots: enabled.powerspot ? showInactivePowerspots : defaults.showInactivePowerspots,
    groupByLayer,
    radiusOverlays: sanitizeRadiusOverlays(obj.radiusOverlays),
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
