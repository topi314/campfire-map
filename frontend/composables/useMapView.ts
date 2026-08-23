import { DEFAULT_MAP_CENTER, DEFAULT_MAP_ZOOM } from "~/constants/map";

const STORAGE_KEY = "campfire-export.mapView";

export interface StoredMapView {
  lat: number;
  lng: number;
  zoom: number;
}

function isValidView(v: StoredMapView) {
  return (
    Number.isFinite(v.lat) &&
    Number.isFinite(v.lng) &&
    Number.isFinite(v.zoom) &&
    v.lat >= -90 &&
    v.lat <= 90 &&
    v.lng >= -180 &&
    v.lng <= 180 &&
    v.zoom >= 0 &&
    v.zoom <= 22
  );
}

export function readStoredMapView(): StoredMapView | null {
  if (!import.meta.client) return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredMapView;
    if (!isValidView(parsed)) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function writeStoredMapView(lat: number, lng: number, zoom: number) {
  if (!import.meta.client) return;
  const view = { lat, lng, zoom };
  if (!isValidView(view)) return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(view));
  } catch {
    /* private mode */
  }
}

export function initialMapView(): StoredMapView {
  const stored = readStoredMapView();
  if (stored) return stored;
  return { lat: DEFAULT_MAP_CENTER[0], lng: DEFAULT_MAP_CENTER[1], zoom: DEFAULT_MAP_ZOOM };
}
