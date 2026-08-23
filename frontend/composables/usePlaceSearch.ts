export interface PlaceResult {
  id: number;
  label: string;
  lat: number;
  lng: number;
  south: number;
  north: number;
  west: number;
  east: number;
}

export async function searchPlaces(
  query: string,
  opts?: { signal?: AbortSignal; apiBase?: string },
): Promise<PlaceResult[]> {
  const q = query.trim();
  if (!q) return [];

  const base = opts?.apiBase ?? "";
  const res = await fetch(`${base}/api/places?q=${encodeURIComponent(q)}`, {
    signal: opts?.signal,
    headers: { Accept: "application/json" },
  });
  if (!res.ok) {
    throw new Error("Place search failed");
  }
  const data = (await res.json()) as { places?: PlaceResult[] };
  return data.places ?? [];
}
