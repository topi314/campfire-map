export default defineNuxtPlugin(async () => {
  const leaflet = await import("leaflet");
  const L = leaflet.default;
  (globalThis as unknown as { L: typeof L }).L = L;
  (window as unknown as { L: typeof L }).L = L;
  await import("leaflet.markercluster");
});
