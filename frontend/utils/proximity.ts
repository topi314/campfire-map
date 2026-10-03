import { WAYFARER_MIN_SEPARATION_M } from "~/constants/map";
import type { Poi } from "~/types/poi";

/** Ray-casting point-in-polygon. Ring points are [lat, lng]. */
export function pointInPolygon(lat: number, lng: number, ring: [number, number][]): boolean {
  if (ring.length < 3) return false;
  let inside = false;
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const yi = ring[i][0];
    const xi = ring[i][1];
    const yj = ring[j][0];
    const xj = ring[j][1];
    const denom = yj - yi || Number.EPSILON;
    const intersect = yi > lat !== yj > lat && lng < ((xj - xi) * (lat - yi)) / denom + xi;
    if (intersect) inside = !inside;
  }
  return inside;
}

/** Approximate meters between two lat/lng points (haversine). */
export function distanceMeters(aLat: number, aLng: number, bLat: number, bLng: number): number {
  const R = 6371000;
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(bLat - aLat);
  const dLng = toRad(bLng - aLng);
  const lat1 = toRad(aLat);
  const lat2 = toRad(bLat);
  const h =
    Math.sin(dLat / 2) ** 2 + Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.min(1, Math.sqrt(h)));
}

export function findNearbyPoi(
  lat: number,
  lng: number,
  others: Poi[],
  excludeId?: string,
  minM = WAYFARER_MIN_SEPARATION_M,
): { poi: Poi; meters: number } | null {
  let best: { poi: Poi; meters: number } | null = null;
  for (const p of others) {
    if (p.id === excludeId) continue;
    if (p.type === "route") continue;
    const meters = distanceMeters(lat, lng, p.lat, p.lng);
    if (meters >= minM) continue;
    if (!best || meters < best.meters) best = { poi: p, meters };
  }
  return best;
}
