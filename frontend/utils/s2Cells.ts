import { pointInPolygon } from "~/utils/proximity";
import type { MapCell, Poi } from "~/types/poi";

const S2_MAX_LEVEL = 30;

/** Gyms appear in an L14 cell at 2, 6, and 20 occupied L17 cells. */
const GYM_AT = [2, 6, 20] as const;

export interface S2Occupancy {
  /** L14 cell id → occupied L17 cells, or pin count when L17 geometry is absent. */
  byL14: Map<string, number>;
  occupiedL17: Set<string>;
  hasL17: boolean;
}

/** Parent S2 cell id at `level`, matching golang/geo CellID.Parent. */
export function parentCellId(id: string, level: number): string {
  const cell = BigInt(id);
  const lsb = 1n << BigInt(2 * (S2_MAX_LEVEL - level));
  return ((cell & -lsb) | lsb).toString();
}

export function occupiesS2Cell(p: Poi): boolean {
  return p.type === "gym" || p.type === "super_mega_gym" || p.type === "pokestop";
}

export function ringCentroid(ring: [number, number][]): [number, number] {
  let n = ring.length;
  if (
    n > 1 &&
    ring[0][0] === ring[n - 1][0] &&
    ring[0][1] === ring[n - 1][1]
  ) {
    n -= 1;
  }
  if (n < 1) return [0, 0];
  let lat = 0;
  let lng = 0;
  for (let i = 0; i < n; i++) {
    lat += ring[i][0];
    lng += ring[i][1];
  }
  return [lat / n, lng / n];
}

function cellHasOccupant(cell: MapCell, pois: Poi[]): boolean {
  for (const p of pois) {
    if (pointInPolygon(p.lat, p.lng, cell.ring)) return true;
  }
  return false;
}

/** Occupancy of gyms and PokéStops. Callers should pass only occupying POIs. */
export function s2Occupancy(cells: MapCell[], pois: Poi[]): S2Occupancy {
  const l17 = cells.filter((c) => c.level === 17 && c.ring && c.ring.length >= 3);
  const byL14 = new Map<string, number>();
  const occupiedL17 = new Set<string>();

  if (l17.length === 0) {
    for (const cell of cells) {
      if (cell.level !== 14 || !cell.ring || cell.ring.length < 3) continue;
      let n = 0;
      for (const p of pois) {
        if (pointInPolygon(p.lat, p.lng, cell.ring)) n++;
      }
      byL14.set(cell.id, n);
    }
    return { byL14, occupiedL17, hasL17: false };
  }

  for (const cell of l17) {
    if (!cellHasOccupant(cell, pois)) continue;
    occupiedL17.add(cell.id);
    const parent = parentCellId(cell.id, 14);
    byL14.set(parent, (byL14.get(parent) ?? 0) + 1);
  }
  return { byL14, occupiedL17, hasL17: true };
}

export interface GymProgress {
  /** Gyms this cell already qualifies for, from 0 to 3. */
  gyms: number;
  /** Occupied cells still needed for the next gym. 0 when the cell is full. */
  remaining: number;
}

export function gymProgress(occupied: number): GymProgress {
  const n = Math.max(0, Math.floor(occupied));
  if (n >= GYM_AT[2]) return { gyms: 3, remaining: 0 };
  if (n >= GYM_AT[1]) return { gyms: 2, remaining: GYM_AT[2] - n };
  if (n >= GYM_AT[0]) return { gyms: 1, remaining: GYM_AT[1] - n };
  return { gyms: 0, remaining: GYM_AT[0] - n };
}
