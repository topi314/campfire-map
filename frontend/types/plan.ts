import {
  DEFAULT_EXPORT_SETTINGS,
  normalizeExportLayers,
  type ExportLayer,
  type ExportSettings,
} from "~/types/export";
import { normalizePoi, type Poi } from "~/types/poi";

export const PLAN_FILE_VERSION = 1;
export const PLAN_FILE_SUFFIX = ".campfire-plan.json";

export interface CampfirePlan {
  version: number;
  mapName: string;
  fileName: string;
  existingPois: Poi[];
  campsitePois: Poi[];
  outline: [number, number][] | null;
  layers: ExportLayer[];
  exportSettings: Omit<ExportSettings, "layers" | "mapName" | "fileName">;
}

export function buildCampfirePlan(input: {
  mapName: string;
  fileName: string;
  existingPois: Poi[];
  campsitePois: Poi[];
  outline: [number, number][] | null;
  settings: ExportSettings;
}): CampfirePlan {
  const { layers, mapName: _m, fileName: _f, ...rest } = input.settings;
  return {
    version: PLAN_FILE_VERSION,
    mapName: input.mapName,
    fileName: input.fileName,
    existingPois: input.existingPois,
    campsitePois: input.campsitePois,
    outline: input.outline,
    layers: normalizeExportLayers(layers),
    exportSettings: rest,
  };
}

export function parseCampfirePlan(raw: unknown): CampfirePlan {
  if (!raw || typeof raw !== "object") throw new Error("Invalid plan file");
  const o = raw as Record<string, unknown>;
  if (o.version !== PLAN_FILE_VERSION) {
    throw new Error(`Unsupported plan version (expected ${PLAN_FILE_VERSION})`);
  }
  if (!Array.isArray(o.existingPois) || !Array.isArray(o.campsitePois)) {
    throw new Error("Plan must include existingPois and campsitePois arrays");
  }
  const existingPois = (o.existingPois as Poi[]).map(normalizePoi);
  const campsitePois = (o.campsitePois as Poi[]).map((p) =>
    normalizePoi({ ...p, source: "campsite" }),
  );
  let outline: [number, number][] | null = null;
  if (o.outline != null) {
    if (!Array.isArray(o.outline)) throw new Error("outline must be an array or null");
    outline = o.outline.map((pt) => {
      if (!Array.isArray(pt) || pt.length < 2) throw new Error("Invalid outline point");
      return [Number(pt[0]), Number(pt[1])] as [number, number];
    });
    if (outline.length < 3) outline = null;
  }
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
    outline,
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
