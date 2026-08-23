export interface BaseMapDefinition {
  id: string;
  label: string;
  url: string;
  attribution: string;
  maxZoom: number;
  subdomains?: string;
}

export const DEFAULT_BASE_MAP_ID = "carto-voyager";

export const BASE_MAPS: BaseMapDefinition[] = [
  {
    id: "carto-voyager",
    label: "Voyager",
    url: "https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png",
    attribution: "&copy; CARTO &copy; OpenStreetMap",
    maxZoom: 20,
    subdomains: "abcd",
  },
  {
    id: "osm",
    label: "OpenStreetMap",
    url: "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
    attribution: "&copy; OpenStreetMap",
    maxZoom: 19,
    subdomains: "abc",
  },
  {
    id: "carto-positron",
    label: "Positron (light)",
    url: "https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png",
    attribution: "&copy; CARTO &copy; OpenStreetMap",
    maxZoom: 20,
    subdomains: "abcd",
  },
  {
    id: "carto-dark",
    label: "Dark matter",
    url: "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png",
    attribution: "&copy; CARTO &copy; OpenStreetMap",
    maxZoom: 20,
    subdomains: "abcd",
  },
  {
    id: "carto-voyager-nolabels",
    label: "Voyager (no labels)",
    url: "https://{s}.basemaps.cartocdn.com/rastertiles/voyager_nolabels/{z}/{x}/{y}{r}.png",
    attribution: "&copy; CARTO &copy; OpenStreetMap",
    maxZoom: 20,
    subdomains: "abcd",
  },
  {
    id: "opentopomap",
    label: "OpenTopoMap",
    url: "https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png",
    attribution:
      "&copy; OpenTopoMap (&copy; OpenStreetMap, SRTM)",
    maxZoom: 17,
    subdomains: "abc",
  },
  {
    id: "esri-street",
    label: "Esri street",
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
  {
    id: "esri-topo",
    label: "Esri topo",
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
  {
    id: "satellite",
    label: "Satellite",
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
  {
    id: "cyclosm",
    label: "CyclOSM",
    url: "https://{s}.tile-cyclosm.openstreetmap.fr/cyclosm/{z}/{x}/{y}.png",
    attribution: "&copy; CyclOSM &copy; OpenStreetMap",
    maxZoom: 20,
    subdomains: "abc",
  },
  {
    id: "wikimedia",
    label: "Wikimedia",
    url: "https://maps.wikimedia.org/osm-intl/{z}/{x}/{y}.png",
    attribution: "&copy; Wikimedia &copy; OpenStreetMap",
    maxZoom: 19,
  },
  {
    id: "esri-hybrid",
    label: "Satellite + labels",
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/Reference/World_Boundaries_and_Places/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
];

/** Satellite imagery with Esri reference labels (two layers). */
export const HYBRID_BASE_LAYERS: Pick<BaseMapDefinition, "url" | "attribution" | "maxZoom" | "subdomains">[] = [
  {
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
  {
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/Reference/World_Transportation/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
  {
    url: "https://server.arcgisonline.com/ArcGIS/rest/services/Reference/World_Boundaries_and_Places/MapServer/tile/{z}/{y}/{x}",
    attribution: "&copy; Esri",
    maxZoom: 19,
  },
];

export function getBaseMap(id: string) {
  const found = BASE_MAPS.find((m) => m.id === id);
  if (found) return found;
  return BASE_MAPS.find((m) => m.id === DEFAULT_BASE_MAP_ID) ?? BASE_MAPS[0];
}

export function isHybridBaseMap(id: string) {
  return id === "esri-hybrid";
}
