<template>
  <div class="app" :class="{ 'export-mode': exportMode }">
    <SettingsModal
      :open="settingsOpen"
      :token="token"
      @close="closeSettings"
      @save="onTokenSave"
      @clear="onTokenClear"
    />
    <TutorialModal :open="tutorialOpen" @close="closeTutorial" />
    <div class="map-wrap">
      <div class="map-search-bar">
        <MapSearch @go="goToPlace" />
      </div>
      <div ref="mapEl" class="map" />
    </div>
    <Sidebar
      v-model:enabled="enabled"
      v-model:base-layer-id="baseLayerId"
      v-model:show-cells="showCells"
      v-model:export-mode="exportMode"
      v-model:show-all-routes="showAllRoutes"
      v-model:group-by-layer="groupByLayer"
      :l14-count="l14Count"
      :l17-count="l17Count"
      :pois="pois"
      :selected="selected"
      :loading="loading"
      :error="error"
      @toggle="onSidebarPoiClick"
      @request-powerspot-auth="onRequestPowerspotAuth"
    />
    <ExportSidebar
      v-if="exportMode"
      v-model:export-mode="exportMode"
      v-model:group-by-layer="groupByLayer"
      :enabled="enabled"
      :pois="pois"
      :export-pois="exportPois"
      :selected="selected"
      :drawing="drawing"
      @toggle="togglePoi"
      @select-ids="setSelection"
      @toggle-draw="toggleDraw"
    />
  </div>
</template>

<script setup lang="ts">
import { initialMapView, writeStoredMapView } from "~/composables/useMapView";
import { readStoredUiSettings, writeStoredUiSettings } from "~/composables/useUiSettings";
import { DEFAULT_MAP_ZOOM } from "~/constants/map";
import { getBaseMap, HYBRID_BASE_LAYERS, isHybridBaseMap } from "~/constants/baseMaps";
import type { PlaceResult } from "~/composables/usePlaceSearch";
import type {
  Map as LeafletMap,
  Marker,
  MarkerClusterGroup,
  FeatureGroup,
  LayerGroup,
  Layer,
  LatLngBounds,
  LeafletMouseEvent,
} from "leaflet";
import {
  ALL_TYPES,
  ROUTE_COLORS,
  TYPE_META,
  normalizePoi,
  poiDisplayType,
  poiLayerVisible,
  type MapCell,
  type Poi,
  type PoiType,
} from "~/types/poi";
import { POI_POPUP_OPTS, poiPopupHtml } from "~/utils/poiPopup";
import { attachExportGestures, attachLongPress, eventHasShift } from "~/utils/exportGesture";

const config = useRuntimeConfig();
const { token, settingsOpen, save, openSettings, closeSettings, clearToken, invalidateToken, authHeaders } =
  useSessionToken();
const { tutorialOpen, closeTutorial, maybeShowTutorial } = useTutorial();
const mapEl = ref<HTMLElement | null>(null);
const pois = ref<Poi[]>([]);
const selected = ref<Set<string>>(new Set());
const exportPoisById = ref<Map<string, Poi>>(new Map());

const storedUi = readStoredUiSettings();
const exportMode = ref(storedUi.exportMode);
const showAllRoutes = ref(storedUi.showAllRoutes);
const groupByLayer = ref(storedUi.groupByLayer);

const exportPois = computed(() => {
  const out: Poi[] = [];
  for (const id of selected.value) {
    const p = exportPoisById.value.get(id);
    if (p) out.push(p);
  }
  return out;
});

const enabled = ref<Record<PoiType, boolean>>({ ...storedUi.enabled });
const loading = ref(false);
const error = ref("");
const drawing = ref(false);
const baseLayerId = ref(storedUi.baseLayerId);
const showCells = ref(storedUi.showCells);
const cells = ref<MapCell[]>([]);
const l14Count = computed(() => cells.value.filter((c) => c.level === 14).length);
const l17Count = computed(() => cells.value.filter((c) => c.level === 17).length);
const expandedRoutes = ref<Set<string>>(new Set());
const focusedRouteId = ref<string | null>(null);

function isLayerEnabled(p: Poi) {
  return poiLayerVisible(p, enabled.value);
}

function displayType(p: Poi) {
  return poiDisplayType(p, enabled.value.super_mega_gym);
}

function persistUiSettings() {
  writeStoredUiSettings({
    enabled: { ...enabled.value },
    baseLayerId: baseLayerId.value,
    showCells: showCells.value,
    exportMode: exportMode.value,
    showAllRoutes: showAllRoutes.value,
    groupByLayer: groupByLayer.value,
  });
}

let map: LeafletMap | null = null;
let baseTileLayer: Layer | null = null;
let cluster: MarkerClusterGroup | null = null;
let routesLayer: LayerGroup | null = null;
let cellsLayer: LayerGroup | null = null;
let drawn: FeatureGroup | null = null;
let stopDrawSession: (() => void) | null = null;
const markers = new Map<string, Marker>();
const routeOverlays = new Map<string, Layer[]>();
const poiCache = new Map<string, Poi>();
let loadTimer: ReturnType<typeof setTimeout> | null = null;
let openPopupId: string | null = null;
let loadSeq = 0;
let suppressPopupClose = false;
let pendingPowerspotEnable = false;

onMounted(async () => {
  const leaflet = await import("leaflet");
  const L = leaflet.default;
  (window as unknown as { L: typeof L }).L = L;
  (globalThis as unknown as { L: typeof L }).L = L;
  await import("leaflet.markercluster");

  if (!mapEl.value) return;
  const start = initialMapView();
  map = L.map(mapEl.value, { zoomControl: true, maxZoom: 20 }).setView([start.lat, start.lng], start.zoom);
  map.getContainer().setAttribute("tabindex", "0");
  applyBaseLayer(baseLayerId.value);

  cluster = L.markerClusterGroup({
    maxClusterRadius: 40,
    disableClusteringAtZoom: 14,
    animate: false,
    chunkedLoading: false,
    spiderfyOnMaxZoom: true,
    zoomToBoundsOnClick: true,
  });
  routesLayer = L.layerGroup();
  cellsLayer = L.layerGroup();
  drawn = L.featureGroup();
  map.addLayer(cellsLayer);
  map.addLayer(cluster);
  map.addLayer(routesLayer);
  map.addLayer(drawn);

  map.on("moveend", () => {
    persistMapView();
    showCachedViewport();
    scheduleLoad();
  });

  if (enabled.value.powerspot && !token.value) {
    enabled.value = { ...enabled.value, powerspot: false };
  }

  await loadPois();
  maybeShowTutorial();
});

onBeforeUnmount(() => {
  stopRectangleDraw();
  map?.remove();
});

watch(
  [enabled, baseLayerId, showCells, exportMode, showAllRoutes, groupByLayer],
  () => persistUiSettings(),
  { deep: true },
);
watch(enabled, () => {
  syncMarkers();
  syncRouteOverlays();
}, { deep: true });
watch(selected, () => {
  syncMarkers();
  refreshMarkerSelection();
});
watch(baseLayerId, (id) => applyBaseLayer(id));
watch(showCells, () => renderCells());
watch(showAllRoutes, (on) => {
  if (!on && focusedRouteId.value) {
    for (const id of [...expandedRoutes.value]) {
      if (id !== focusedRouteId.value) hideRouteOverlay(id);
    }
    expandedRoutes.value = new Set([focusedRouteId.value]);
  }
  syncRouteOverlays();
});
watch(exportMode, (on) => {
  if (!on) {
    if (drawing.value) drawn?.clearLayers();
    stopRectangleDraw();
  }
  nextTick(() => {
    map?.invalidateSize();
  });
});
watch(settingsOpen, (open) => {
  if (!open && !token.value) {
    pendingPowerspotEnable = false;
  }
});

function applyBaseLayer(id: string) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L) return;
  if (baseTileLayer) {
    map.removeLayer(baseTileLayer);
    baseTileLayer = null;
  }
  if (isHybridBaseMap(id)) {
    const group = L.layerGroup();
    for (const layer of HYBRID_BASE_LAYERS) {
      L.tileLayer(layer.url, {
        attribution: layer.attribution,
        maxZoom: layer.maxZoom,
        subdomains: layer.subdomains ?? "abc",
      }).addTo(group);
    }
    group.addTo(map);
    baseTileLayer = group;
    return;
  }
  const def = getBaseMap(id);
  baseTileLayer = L.tileLayer(def.url, {
    attribution: def.attribution,
    maxZoom: def.maxZoom,
    subdomains: def.subdomains ?? "abc",
  });
  baseTileLayer.addTo(map);
}

function stopRectangleDraw() {
  stopDrawSession?.();
  stopDrawSession = null;
  drawing.value = false;
}

function selectedIdsInBounds(bounds: LatLngBounds) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!L) return [] as string[];
  const ids: string[] = [];
  const seen = new Set<string>();
  const consider = (p: Poi) => {
    if (seen.has(p.id) || !isLayerEnabled(p)) return;
    if (!bounds.contains(L.latLng(p.lat, p.lng))) return;
    seen.add(p.id);
    ids.push(p.id);
  };
  for (const p of pois.value) consider(p);
  for (const p of poiCache.values()) consider(p);
  return ids;
}

function startRectangleDraw() {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L || drawing.value) return;

  drawing.value = true;
  drawn?.clearLayers();
  map.getContainer().focus();

  const wasDragging = map.dragging.enabled();
  if (wasDragging) map.dragging.disable();
  map.getContainer().style.cursor = "crosshair";

  let startLatLng: import("leaflet").LatLng | null = null;
  let rect: import("leaflet").Rectangle | null = null;
  let dragged = false;

  const onMouseDown = (e: LeafletMouseEvent) => {
    L.DomEvent.preventDefault(e.originalEvent);
    startLatLng = e.latlng;
    dragged = false;
    drawn?.clearLayers();
    rect = L.rectangle(L.latLngBounds(startLatLng, startLatLng), {
      color: "#5b8cff",
      weight: 2,
      fillOpacity: 0.15,
    });
    drawn?.addLayer(rect);
  };

  const onMouseMove = (e: LeafletMouseEvent) => {
    if (!startLatLng || !rect) return;
    dragged = true;
    rect.setBounds(L.latLngBounds(startLatLng, e.latlng));
  };

  const finishShape = (end: import("leaflet").LatLng) => {
    if (!startLatLng) return;
    const origin = startLatLng;
    startLatLng = null;
    const bounds = L.latLngBounds(origin, end);
    if (dragged && bounds.isValid()) {
      const next = new Set(selected.value);
      for (const id of selectedIdsInBounds(bounds)) next.add(id);
      setSelection([...next]);
    }
    drawn?.clearLayers();
    rect = null;
    dragged = false;
  };

  const onMouseUp = (e: LeafletMouseEvent) => {
    if (!startLatLng) return;
    finishShape(e.latlng);
  };

  const onDocMouseUp = (ev: MouseEvent) => {
    if (!startLatLng || !map) return;
    finishShape(map.mouseEventToLatLng(ev));
  };

  const onKeyDown = (ev: KeyboardEvent) => {
    if (ev.key === "Escape") {
      drawn?.clearLayers();
      stopRectangleDraw();
    }
  };

  map.on("mousedown", onMouseDown);
  map.on("mousemove", onMouseMove);
  map.on("mouseup", onMouseUp);
  document.addEventListener("mouseup", onDocMouseUp);
  document.addEventListener("keydown", onKeyDown);

  stopDrawSession = () => {
    map?.off("mousedown", onMouseDown);
    map?.off("mousemove", onMouseMove);
    map?.off("mouseup", onMouseUp);
    document.removeEventListener("mouseup", onDocMouseUp);
    document.removeEventListener("keydown", onKeyDown);
    if (wasDragging) map?.dragging.enable();
    if (map) map.getContainer().style.cursor = "";
  };
}

function toggleDraw() {
  if (!map) return;
  if (drawing.value) {
    drawn?.clearLayers();
    stopRectangleDraw();
    return;
  }
  startRectangleDraw();
}

function togglePoi(p: Poi) {
  if (!exportMode.value) {
    exportMode.value = true;
  }
  const next = new Set(selected.value);
  if (next.has(p.id)) {
    next.delete(p.id);
    forgetExportPoi(p.id);
  } else {
    next.add(p.id);
    rememberExportPoi(p);
  }
  selected.value = next;
}

function onSidebarPoiClick(p: Poi, ev?: MouseEvent, exportSelect = false) {
  if (exportSelect || ev?.shiftKey) {
    togglePoi(p);
  }
  focusPoi(p);
}

function focusPoi(p: Poi) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L || !cluster) return;
  const poi = resolvePoi(p);

  if (poi.type === "route" && poi.path && poi.path.length > 1) {
    expandRoute(poi);
    const bounds = L.latLngBounds(poi.path.map(([lat, lng]) => L.latLng(lat, lng)));
    map.fitBounds(bounds, { padding: [48, 48], maxZoom: 18 });
    const marker = markers.get(poi.id);
    if (!marker) return;
    map.once("moveend", () => {
      cluster?.zoomToShowLayer(marker, () => {
        marker.openPopup();
      });
    });
    return;
  }

  const marker = markers.get(poi.id);
  if (marker) {
    cluster.zoomToShowLayer(marker, () => {
      marker.openPopup();
    });
    return;
  }

  map.setView([poi.lat, poi.lng], Math.max(map.getZoom(), DEFAULT_MAP_ZOOM));
}

function resolvePoi(p: Poi) {
  return poiCache.get(p.id) ?? exportPoisById.value.get(p.id) ?? p;
}

function attachExportSelect(layer: Layer, p: Poi) {
  attachExportGestures(layer, () => togglePoi(resolvePoi(p)));
}

function setSelection(ids: string[]) {
  const next = new Set(ids);
  const byId = new Map(exportPoisById.value);
  for (const id of byId.keys()) {
    if (!next.has(id)) byId.delete(id);
  }
  const viewportById = new Map(pois.value.map((poi) => [poi.id, poi]));
  for (const id of next) {
    const p = viewportById.get(id) ?? byId.get(id) ?? poiCache.get(id);
    if (p) byId.set(id, p);
  }
  exportPoisById.value = byId;
  selected.value = next;
}

function rememberExportPoi(p: Poi) {
  exportPoisById.value = new Map(exportPoisById.value).set(p.id, p);
}

function forgetExportPoi(id: string) {
  if (!exportPoisById.value.has(id)) return;
  const next = new Map(exportPoisById.value);
  next.delete(id);
  exportPoisById.value = next;
}

function refreshExportPoisFromViewport() {
  if (exportPoisById.value.size === 0) return;
  const next = new Map(exportPoisById.value);
  for (const p of pois.value) {
    if (selected.value.has(p.id)) {
      next.set(p.id, p);
    }
  }
  exportPoisById.value = next;
}

function persistMapView() {
  if (!map) return;
  const center = map.getCenter();
  writeStoredMapView(center.lat, center.lng, map.getZoom());
}

function scheduleLoad() {
  if (loadTimer) clearTimeout(loadTimer);
  loadTimer = setTimeout(() => {
    loadPois();
  }, 400);
}

function onRequestPowerspotAuth() {
  pendingPowerspotEnable = true;
  openSettings();
}

async function onTokenSave(next: string) {
  if (!save(next)) return;
  if (pendingPowerspotEnable) {
    pendingPowerspotEnable = false;
    enabled.value = { ...enabled.value, powerspot: true };
  }
  await loadPois();
}

function onTokenClear() {
  clearToken();
  pendingPowerspotEnable = false;
  enabled.value = { ...enabled.value, powerspot: false };
  for (const [id, p] of [...poiCache.entries()]) {
    if (p.type === "powerspot") poiCache.delete(id);
  }
  const nextSelected = new Set(selected.value);
  const nextExport = new Map(exportPoisById.value);
  for (const [id, p] of nextExport) {
    if (p.type === "powerspot") {
      nextSelected.delete(id);
      nextExport.delete(id);
    }
  }
  exportPoisById.value = nextExport;
  selected.value = nextSelected;
  if (map) {
    pois.value = poisInBounds(map.getBounds());
  }
  syncMarkers();
  scheduleLoad();
}

function goToPlace(place: PlaceResult) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L) return;
  const sw = L.latLng(place.south, place.west);
  const ne = L.latLng(place.north, place.east);
  if (place.south !== place.north || place.west !== place.east) {
    map.fitBounds(L.latLngBounds(sw, ne), { maxZoom: 17, padding: [40, 40] });
  } else {
    map.setView([place.lat, place.lng], DEFAULT_MAP_ZOOM);
  }
}

function requestTypes(): PoiType[] {
  if (token.value) return ALL_TYPES;
  return ALL_TYPES.filter((t) => t !== "powerspot");
}

async function loadPois() {
  if (!map) return;
  const seq = ++loadSeq;
  const b = map.getBounds();
  const bbox = [b.getSouth(), b.getWest(), b.getNorth(), b.getEast()].join(",");
  loading.value = true;
  error.value = "";
  try {
    const types = requestTypes().join(",");
    const res = await fetch(
      `${config.public.apiBase}/api/pois?bbox=${encodeURIComponent(bbox)}&types=${encodeURIComponent(types)}`,
      { headers: authHeaders() },
    );
    if (!res.ok) {
      if (res.status === 401 || res.status === 403) {
        invalidateToken();
        enabled.value = { ...enabled.value, powerspot: false };
        throw new Error("Session token rejected. Paste a fresh Campfire token.");
      }
      throw new Error(await res.text());
    }
    const data = await res.json();
    if (seq !== loadSeq) return;
    for (const p of (data.pois || []) as Poi[]) {
      const n = normalizePoi(p);
      poiCache.set(n.id, n);
    }
    prunePoiCache(b);
    cells.value = data.cells || [];
    pois.value = poisInBounds(b);
    refreshExportPoisFromViewport();
    renderCells();
    syncMarkers();
  } catch (e) {
    if (seq !== loadSeq) return;
    error.value = e instanceof Error ? e.message : "Failed to load POIs";
  } finally {
    if (seq === loadSeq) loading.value = false;
  }
}

function poisInBounds(b: LatLngBounds) {
  const out: Poi[] = [];
  for (const p of poiCache.values()) {
    if (b.contains([p.lat, p.lng])) out.push(p);
  }
  return out;
}

function showCachedViewport() {
  if (!map || poiCache.size === 0) return;
  pois.value = poisInBounds(map.getBounds());
  syncMarkers();
}

function prunePoiCache(b: LatLngBounds) {
  if (poiCache.size <= 12000) return;
  const padLat = (b.getNorth() - b.getSouth()) * 2;
  const padLng = (b.getEast() - b.getWest()) * 2;
  const south = b.getSouth() - padLat;
  const north = b.getNorth() + padLat;
  const west = b.getWest() - padLng;
  const east = b.getEast() + padLng;
  for (const [id, p] of poiCache) {
    if (selected.value.has(id)) continue;
    if (p.lat < south || p.lat > north || p.lng < west || p.lng > east) {
      poiCache.delete(id);
    }
  }
}

function attachPoiPopup(layer: Layer, p: Poi) {
  const id = p.id;
  layer.bindPopup(() => {
    const cur = poiCache.get(id) ?? exportPoisById.value.get(id) ?? p;
    return poiPopupHtml(cur, selected.value.has(id), displayType(cur));
  }, POI_POPUP_OPTS);
  layer.on("popupopen", () => {
    openPopupId = id;
    bindPopupToggle(layer, poiCache.get(id) ?? exportPoisById.value.get(id) ?? p);
  });
  layer.on("popupclose", () => {
    if (suppressPopupClose) return;
    if (openPopupId === id) openPopupId = null;
  });
}

function bindPopupToggle(layer: Layer, p: Poi) {
  const popup = layer.getPopup();
  const btn = popup?.getElement()?.querySelector("[data-poi-toggle]");
  if (!btn || (btn as HTMLElement).dataset.bound === "1") return;
  (btn as HTMLElement).dataset.bound = "1";
  btn.addEventListener("click", (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    togglePoi(p);
    suppressPopupClose = true;
    try {
      popup?.setContent(poiPopupHtml(p, selected.value.has(p.id), displayType(p)));
    } finally {
      suppressPopupClose = false;
    }
    bindPopupToggle(layer, p);
  });
}

function routeStartLatLng(L: typeof import("leaflet").default, p: Poi) {
  if (p.path && p.path.length > 0) {
    const [lat, lng] = p.path[0];
    return L.latLng(lat, lng);
  }
  return L.latLng(p.lat, p.lng);
}

function expandRoute(p: Poi) {
  if (!p.path || p.path.length <= 1) return;
  const prevFocus = focusedRouteId.value;
  focusedRouteId.value = p.id;

  if (!showAllRoutes.value) {
    for (const id of [...expandedRoutes.value]) {
      if (id === p.id) continue;
      hideRouteOverlay(id);
    }
    expandedRoutes.value = new Set([p.id]);
  } else {
    const next = new Set(expandedRoutes.value);
    next.add(p.id);
    expandedRoutes.value = next;
    if (prevFocus && prevFocus !== p.id) {
      const prev = poiCache.get(prevFocus) ?? exportPoisById.value.get(prevFocus);
      if (prev?.path && prev.path.length > 1) showRouteOverlay(prev);
    }
  }

  showRouteOverlay(p);
  const layers = routeOverlays.get(p.id);
  if (layers) {
    for (const layer of layers) {
      if ("bringToFront" in layer && typeof layer.bringToFront === "function") {
        layer.bringToFront();
      }
    }
  }
}

function collapseRoute(id: string) {
  if (showAllRoutes.value) {
    if (focusedRouteId.value !== id) return;
    focusedRouteId.value = null;
    const p = poiCache.get(id) ?? exportPoisById.value.get(id);
    if (p?.path && p.path.length > 1 && enabled.value.route) showRouteOverlay(p);
    return;
  }
  if (!expandedRoutes.value.has(id)) return;
  const next = new Set(expandedRoutes.value);
  next.delete(id);
  expandedRoutes.value = next;
  if (focusedRouteId.value === id) focusedRouteId.value = null;
  hideRouteOverlay(id);
}

function hideRouteOverlay(id: string) {
  const layers = routeOverlays.get(id);
  if (!layers || !routesLayer) return;
  for (const layer of layers) {
    routesLayer.removeLayer(layer);
  }
  routeOverlays.delete(id);
}

function showRouteOverlay(p: Poi) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!L || !routesLayer || !p.path || p.path.length <= 1) return;
  hideRouteOverlay(p.id);

  const latlngs = p.path.map(([lat, lng]) => L.latLng(lat, lng));
  const exportSelected = selected.value.has(p.id);
  const clickFocused = focusedRouteId.value === p.id;
  const highlighted = exportSelected || clickFocused;
  const strokeWeight = 5;
  const strokeColor = highlighted ? "#e040fb" : ROUTE_COLORS.line;
  const layers: Layer[] = [];

  const border = L.polyline(latlngs, {
    color: "#ffffff",
    weight: strokeWeight + 4,
    opacity: highlighted ? 1 : 0.9,
    lineCap: "round",
    lineJoin: "round",
    interactive: false,
  });
  border.addTo(routesLayer);
  layers.push(border);

  const line = L.polyline(latlngs, {
    color: strokeColor,
    weight: strokeWeight,
    opacity: highlighted ? 1 : 0.85,
    lineCap: "round",
    lineJoin: "round",
    interactive: false,
  });
  line.addTo(routesLayer);
  layers.push(line);

  // Wider invisible stroke so the path is easy to click.
  const hit = L.polyline(latlngs, {
    color: strokeColor,
    weight: strokeWeight + 12,
    opacity: 0,
    lineCap: "round",
    lineJoin: "round",
    interactive: true,
  });
  hit.addTo(routesLayer);
  attachPoiPopup(hit, p);
  hit.on("click", (ev) => {
    L.DomEvent.stopPropagation(ev);
    onRoutePartClick(p, ev as LeafletMouseEvent);
  });
  hit.on("add", () => {
    const el = hit.getElement();
    if (el && el.dataset.exportGesture !== "1") {
      el.dataset.exportGesture = "1";
      attachLongPress(el, () => onRoutePartClick(p, undefined, true));
    }
  });
  const hitEl = hit.getElement();
  if (hitEl && hitEl.dataset.exportGesture !== "1") {
    hitEl.dataset.exportGesture = "1";
    attachLongPress(hitEl, () => onRoutePartClick(p, undefined, true));
  }
  layers.push(hit);

  if (highlighted) {
    border.bringToFront();
    line.bringToFront();
    hit.bringToFront();
  }

  const endLl = latlngs[latlngs.length - 1];
  if (latlngs[0].distanceTo(endLl) > 4) {
    const end = L.marker(endLl, {
      icon: routeEndIcon(L, exportSelected),
      zIndexOffset: highlighted ? 1200 : 800,
    });
    end.addTo(routesLayer);
    attachRouteEndpoint(end, p);
    layers.push(end);
  }

  routeOverlays.set(p.id, layers);
}

function onRoutePartClick(p: Poi, ev?: LeafletMouseEvent, exportSelect = false) {
  const cur = resolvePoi(p);
  if (!cur.path || cur.path.length <= 1) return;
  const select = exportSelect || (!!ev && eventHasShift(ev));
  // Rebuilding the overlay removes the end marker; suppress popupclose→collapse.
  suppressPopupClose = true;
  try {
    expandRoute(cur);
  } finally {
    suppressPopupClose = false;
  }
  if (exportMode.value || select) {
    togglePoi(cur);
  }
  if (select) {
    markers.get(cur.id)?.closePopup();
    return;
  }
  // Overlay rebuild destroys path/end layers from this click; open on the start marker after.
  nextTick(() => {
    requestAnimationFrame(() => {
      openRouteInfoPopup(cur.id);
    });
  });
}

function openRouteInfoPopup(id: string) {
  const start = markers.get(id);
  if (start) {
    start.openPopup();
    return;
  }
  const overlay = routeOverlays.get(id);
  if (!overlay) return;
  for (let i = overlay.length - 1; i >= 0; i--) {
    const layer = overlay[i];
    if (layer.getPopup()) {
      layer.openPopup();
      return;
    }
  }
}

function attachRouteEndpoint(marker: Marker, p: Poi) {
  // No popup on the end pin — opening/closing it during overlay rebuild was
  // collapsing the route. Info opens on the start marker via onRoutePartClick.
  marker.on("click", (ev) => {
    const Lany = (window as unknown as { L?: typeof import("leaflet").default }).L;
    Lany?.DomEvent.stopPropagation(ev);
    onRoutePartClick(p, ev as LeafletMouseEvent);
  });
  attachExportGestures(marker, () => onRoutePartClick(p, undefined, true), { shiftClick: false });
}

function attachRouteMarker(marker: Marker, p: Poi) {
  attachPoiPopup(marker, p);
  marker.on("click", (ev) => {
    onRoutePartClick(p, ev as LeafletMouseEvent);
  });
  attachExportGestures(marker, () => onRoutePartClick(p, undefined, true), { shiftClick: false });
  marker.on("popupclose", () => {
    if (suppressPopupClose) return;
    collapseRoute(p.id);
  });
}

function renderCells() {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!cellsLayer || !L) return;
  cellsLayer.clearLayers();
  if (!showCells.value) return;
  const ordered = [...cells.value].sort((a, b) => (b.level ?? 0) - (a.level ?? 0));
  for (const cell of ordered) {
    if (!cell.ring || cell.ring.length < 3) continue;
    const latlngs = cell.ring.map(([lat, lng]) => L.latLng(lat, lng));
    const l17 = cell.level === 17;
    L.polygon(latlngs, {
      color: l17 ? "#9eb6ff" : "#5b8cff",
      weight: l17 ? 1 : 2,
      opacity: l17 ? 0.55 : 0.9,
      fillColor: "#5b8cff",
      fillOpacity: l17 ? 0.03 : 0.06,
      interactive: false,
    }).addTo(cellsLayer);
  }
}

function syncMarkers() {
  const L = (window as unknown as { L: typeof import("leaflet").default }).L;
  if (!cluster || !routesLayer || !L) return;
  const desired = new Map<string, Poi>();
  for (const p of pois.value) {
    if (isLayerEnabled(p)) desired.set(p.id, p);
  }
  for (const [id, p] of exportPoisById.value) {
    if (!selected.value.has(id)) continue;
    if (!isLayerEnabled(p)) continue;
    if (!desired.has(id)) desired.set(id, p);
  }

  for (const [id, marker] of markers) {
    if (desired.has(id)) continue;
    cluster.removeLayer(marker);
    markers.delete(id);
    hideRouteOverlay(id);
    if (openPopupId === id) openPopupId = null;
  }

  for (const p of desired.values()) {
    const existing = markers.get(p.id);
    if (existing) {
      refreshMarker(existing, p);
      continue;
    }
    if (p.type === "route") {
      const marker = L.marker(routeStartLatLng(L, p), {
        icon: pinIcon(L, p),
        title: p.name,
      });
      attachRouteMarker(marker, p);
      markers.set(p.id, marker);
      cluster.addLayer(marker);
      continue;
    }
    const marker = L.marker([p.lat, p.lng], {
      icon: pinIcon(L, p),
      title: p.name,
    });
    attachPoiPopup(marker, p);
    attachExportSelect(marker, p);
    markers.set(p.id, marker);
    cluster.addLayer(marker);
  }

  syncRouteOverlays(desired);
  restoreOpenPopup();
}

function syncRouteOverlays(desired?: Map<string, Poi>) {
  if (!enabled.value.route) {
    for (const id of [...routeOverlays.keys()]) hideRouteOverlay(id);
    return;
  }

  const source = desired ?? new Map<string, Poi>();
  if (!desired) {
    for (const p of pois.value) {
      if (isLayerEnabled(p)) source.set(p.id, p);
    }
    for (const [id, p] of exportPoisById.value) {
      if (selected.value.has(id) && isLayerEnabled(p) && !source.has(id)) source.set(id, p);
    }
  }

  const keep = new Set<string>();
  if (showAllRoutes.value) {
    for (const p of source.values()) {
      if (p.type !== "route" || !p.path || p.path.length <= 1) continue;
      keep.add(p.id);
      showRouteOverlay(p);
    }
  } else {
    for (const id of expandedRoutes.value) {
      const p = source.get(id) ?? poiCache.get(id) ?? exportPoisById.value.get(id);
      if (p && p.type === "route" && p.path && p.path.length > 1) {
        keep.add(id);
        showRouteOverlay(p);
      }
    }
  }

  for (const id of [...routeOverlays.keys()]) {
    if (!keep.has(id)) hideRouteOverlay(id);
  }

  for (const id of keep) {
    if (!selected.value.has(id) && focusedRouteId.value !== id) continue;
    const layers = routeOverlays.get(id);
    if (!layers) continue;
    for (const layer of layers) {
      if ("bringToFront" in layer && typeof layer.bringToFront === "function") {
        layer.bringToFront();
      }
    }
  }
}

function refreshMarker(marker: Marker, p: Poi) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (L) {
    marker.setIcon(pinIcon(L, p));
  }
  const selectedNow = selected.value.has(p.id);
  const el = marker.getElement()?.querySelector(".poi-marker");
  if (el) el.classList.toggle("selected-pin", selectedNow);
  const popup = marker.getPopup();
  if (popup?.isOpen()) {
    suppressPopupClose = true;
    try {
      popup.setContent(poiPopupHtml(p, selectedNow, displayType(p)));
    } finally {
      suppressPopupClose = false;
    }
    bindPopupToggle(marker, p);
  }
}

function refreshMarkerSelection() {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!L) return;
  for (const [id, marker] of markers) {
    const p = poiCache.get(id) ?? exportPoisById.value.get(id) ?? pois.value.find((poi) => poi.id === id);
    if (!p) continue;
    const selectedNow = selected.value.has(id);
    marker.setIcon(pinIcon(L, p));
    const el = marker.getElement()?.querySelector(".poi-marker");
    if (el) el.classList.toggle("selected-pin", selectedNow);
    const popup = marker.getPopup();
    if (popup?.isOpen()) {
      suppressPopupClose = true;
      try {
        popup.setContent(poiPopupHtml(p, selectedNow, displayType(p)));
      } finally {
        suppressPopupClose = false;
      }
      bindPopupToggle(marker, p);
    }
  }
  syncRouteOverlays();
}

function restoreOpenPopup() {
  if (!openPopupId) return;
  const marker = markers.get(openPopupId);
  if (!marker) return;
  if (marker.getPopup()?.isOpen()) return;
  marker.openPopup();
}

function routeEndIcon(L: typeof import("leaflet").default, selected: boolean) {
  const sel = selected ? " selected-pin" : "";
  const image = "/images/pgo-route.svg";
  const inner = `<span class="poi-glyph" style="background:${ROUTE_COLORS.end};-webkit-mask-image:url(${image});mask-image:url(${image})"></span>`;
  return L.divIcon({
    className: "poi-icon-wrap route-endpoint-icon route-endpoint-end",
    html: `<div class="poi-marker route-endpoint route-endpoint-end${sel}">${inner}</div>`,
    iconSize: [32, 32],
    iconAnchor: [16, 16],
  });
}

function pinIcon(L: typeof import("leaflet").default, p: Poi) {
  const type = displayType(p);
  const meta = TYPE_META[type];
  const sel = selected.value.has(p.id) ? " selected-pin" : "";
  const inner =
    type === "super_mega_gym"
      ? `<img src="${meta.image}" alt="">`
      : `<span class="poi-glyph" style="background:${meta.color};-webkit-mask-image:url(${meta.image});mask-image:url(${meta.image})"></span>`;
  return L.divIcon({
    className: "poi-icon-wrap",
    html: `<div class="poi-marker${sel}">${inner}</div>`,
    iconSize: [32, 32],
    iconAnchor: [16, 30],
  });
}
</script>
