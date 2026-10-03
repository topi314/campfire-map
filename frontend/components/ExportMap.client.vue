<template>
  <div
    class="app"
    :class="{
      'export-mode': exportMode,
      'campsite-edit': campsiteEdit,
      'sheet-tab-explore': sheetTab === 'explore',
      'sheet-tab-export': sheetTab === 'export',
    }"
  >
    <SettingsModal
      :open="settingsOpen"
      :token="token"
      :wayfarer-enabled="wayfarerEnabled"
      :wayfarer-session="wayfarerSession"
      :wayfarer-xsrf="wayfarerXsrf"
      @close="closeSettings"
      @save="onTokenSave"
      @clear="onTokenClear"
      @save-wayfarer="onWayfarerSave"
      @clear-wayfarer="onWayfarerClear"
    />
    <TutorialModal :open="tutorialOpen" @close="closeTutorial" />
    <div class="map-wrap">
      <div class="map-search-bar">
        <MapSearch @go="goToPlace" />
      </div>
      <div ref="mapEl" class="map" />
    </div>
    <BottomSheet @snap="onSheetSnap">
      <div class="sheet-tabs">
        <button type="button" :class="{ primary: sheetTab === 'explore' }" @click="sheetTab = 'explore'">Explore</button>
        <button type="button" :class="{ primary: sheetTab === 'export' }" @click="openExportSheet">
          Export
          <span class="counts">({{ selected.size + campsitePois.length }})</span>
        </button>
      </div>
      <aside class="sidebar panel-explore">
        <Sidebar
          v-model:enabled="enabled"
          v-model:base-layer-id="baseLayerId"
          v-model:show-cells="showCells"
          v-model:export-mode="exportMode"
          v-model:show-all-routes="showAllRoutes"
          v-model:show-inactive-powerspots="showInactivePowerspots"
          v-model:group-by-layer="groupByLayer"
          v-model:radius-overlays="radiusOverlays"
          :l14-count="l14Count"
          :l17-count="l17Count"
          :pois="displayPois"
          :selected="selected"
          :loading="loading"
          :error="error"
          @toggle="onSidebarPoiClick"
          @request-powerspot-auth="onRequestPowerspotAuth"
        />
      </aside>
      <aside v-show="exportMode" class="sidebar export-sidebar panel-export">
        <ExportSidebar
          v-model:export-mode="exportMode"
          v-model:group-by-layer="groupByLayer"
          :enabled="enabled"
          :show-inactive-powerspots="showInactivePowerspots"
          :pois="displayPois"
          :export-pois="exportPois"
          :campsite-pois="campsitePois"
          :selected="selected"
          :campsite-edit="campsiteEdit"
          :place-tool="placeTool"
          :outline="outline"
          :proximity-warning="proximityWarning"
          :campsite-note="campsiteNote"
          v-model:radius-overlays="radiusOverlays"
          @toggle="togglePoi"
          @enter-campsite="enterCampsiteEdit"
          @exit-campsite="exitCampsiteEdit"
          @set-place-tool="setPlaceTool"
          @clear-outline="clearOutline"
          @clear-selection="clearSelection"
          @exclude-campsite-pois="excludeCampsiteLabeledPois"
          @convert-campsite-pois="convertCampsiteLabeledToPlanned"
          @remove-campsite-poi="removeCampsitePoi"
          @focus-poi="focusPoi"
          @import-plan="importPlan"
        />
      </aside>
    </BottomSheet>
  </div>
</template>

<script setup lang="ts">
import { initialMapView, writeStoredMapView } from "~/composables/useMapView";
import { readStoredUiSettings, writeStoredUiSettings } from "~/composables/useUiSettings";
import {
  CAMPSITE_CA_SOFT_LIMIT,
  DEFAULT_MAP_ZOOM,
  RADIUS_OVERLAYS,
  WAYFARER_MIN_SEPARATION_M,
  anyRadiusOverlayEnabled,
  radiusOverlayApplies,
  type RadiusOverlayDef,
  type RadiusOverlayState,
} from "~/constants/map";
import { getBaseMap, HYBRID_BASE_LAYERS, isHybridBaseMap, withCartoApiKey } from "~/constants/baseMaps";
import type { PlaceResult } from "~/composables/usePlaceSearch";
import type { CampfirePlan } from "~/types/plan";
import {
  DEFAULT_EXPORT_SETTINGS,
  normalizeExportLayers,
  type ExportSettings,
  type PlaceTool,
} from "~/types/export";
import type {
  Map as LeafletMap,
  Marker,
  MarkerClusterGroup,
  FeatureGroup,
  LayerGroup,
  Layer,
  LatLngBounds,
  LeafletMouseEvent,
  Polygon,
  Polyline,
  CircleMarker,
} from "leaflet";
import {
  ALL_TYPES,
  CAMPSITE_MARKER,
  INACTIVE_POWERSPOT,
  ROUTE_COLORS,
  TYPE_META,
  isCampsiteLabeledPoi,
  isInactivePowerspot,
  normalizePoi,
  poiDisplayType,
  poiLayerVisible,
  type MapCell,
  type Poi,
  type PoiType,
} from "~/types/poi";
import { POI_POPUP_OPTS, poiPopupHtml } from "~/utils/poiPopup";
import { attachExportGestures, attachLongPress, eventHasShift } from "~/utils/exportGesture";
import { findNearbyPoi, pointInPolygon } from "~/utils/proximity";

const config = useRuntimeConfig();
const { token, settingsOpen, save, openSettings, closeSettings, clearToken, invalidateToken, authHeaders } =
  useSessionToken();
const {
  enabled: wayfarerEnabled,
  session: wayfarerSession,
  xsrfToken: wayfarerXsrf,
  ready: wayfarerReady,
  save: saveWayfarer,
  clear: clearWayfarer,
  invalidate: invalidateWayfarer,
  wayfarerHeaders,
} = useWayfarerCredentials();
const { apiKey: cartoApiKey } = useCartoApiKey();
const { tutorialOpen, closeTutorial, maybeShowTutorial } = useTutorial();
const mapEl = ref<HTMLElement | null>(null);
const pois = ref<Poi[]>([]);
const selected = ref<Set<string>>(new Set());
const exportPoisById = ref<Map<string, Poi>>(new Map());

const storedUi = readStoredUiSettings();
const exportMode = ref(storedUi.exportMode);
const showAllRoutes = ref(storedUi.showAllRoutes);
const showInactivePowerspots = ref(storedUi.showInactivePowerspots);
const groupByLayer = ref(storedUi.groupByLayer);
const sheetTab = ref<"explore" | "export">(exportMode.value ? "export" : "explore");
const campsiteEdit = ref(false);
const campsitePois = ref<Poi[]>([]);
const outline = ref<[number, number][] | null>(null);
const placeTool = ref<PlaceTool>(null);
const proximityWarning = ref("");
const campsiteNote = ref("");
const radiusOverlays = ref<RadiusOverlayState>({ ...storedUi.radiusOverlays });
const outlineDraft = ref<[number, number][]>([]);
const campsiteConflicts = ref<Set<string>>(new Set());
let campsiteNoteTimer: ReturnType<typeof setTimeout> | null = null;

const exportPois = computed(() => {
  const out: Poi[] = [];
  for (const id of selected.value) {
    const p = exportPoisById.value.get(id);
    if (p) out.push(p);
  }
  return out;
});

const displayPois = computed(() => {
  if (!campsiteEdit.value) return pois.value;
  return [...exportPois.value, ...campsitePois.value];
});

const enabled = ref<Record<PoiType, boolean>>({ ...storedUi.enabled });
const loading = ref(false);
const error = ref("");
const baseLayerId = ref(storedUi.baseLayerId);
const showCells = ref(storedUi.showCells);
const cells = ref<MapCell[]>([]);
const l14Count = computed(() => cells.value.filter((c) => c.level === 14).length);
const l17Count = computed(() => cells.value.filter((c) => c.level === 17).length);
const expandedRoutes = ref<Set<string>>(new Set());
const focusedRouteId = ref<string | null>(null);

function isLayerEnabled(p: Poi) {
  return poiLayerVisible(p, enabled.value, {
    showInactivePowerspots: showInactivePowerspots.value,
  });
}

function displayType(p: Poi) {
  return poiDisplayType(p, enabled.value.super_mega_gym);
}

function canLoadPowerspots() {
  return !!token.value || wayfarerReady.value;
}

function persistUiSettings() {
  writeStoredUiSettings({
    enabled: { ...enabled.value },
    baseLayerId: baseLayerId.value,
    showCells: showCells.value,
    exportMode: exportMode.value,
    showAllRoutes: showAllRoutes.value,
    showInactivePowerspots: showInactivePowerspots.value,
    groupByLayer: groupByLayer.value,
    radiusOverlays: { ...radiusOverlays.value },
  });
}

let map: LeafletMap | null = null;
let baseTileLayer: Layer | null = null;
let cluster: MarkerClusterGroup | null = null;
let campsiteLayer: LayerGroup | null = null;
let routesLayer: LayerGroup | null = null;
let cellsLayer: LayerGroup | null = null;
let outlineLayer: FeatureGroup | null = null;
let proximityRadiiLayer: LayerGroup | null = null;
let outlineShape: Polyline | Polygon | null = null;
let outlineFill: Polygon | null = null;
let outlineDragging = false;
let stopPlaceSession: (() => void) | null = null;
const markers = new Map<string, Marker>();
const routeOverlays = new Map<string, Layer[]>();
const poiCache = new Map<string, Poi>();
let loadTimer: ReturnType<typeof setTimeout> | null = null;
let openPopupId: string | null = null;
let loadSeq = 0;
let suppressPopupClose = false;
let pendingPowerspotEnable = false;
let warnCircle: CircleMarker | null = null;
let draggingCampsiteId: string | null = null;

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
  campsiteLayer = L.layerGroup();
  outlineLayer = L.featureGroup();
  proximityRadiiLayer = L.layerGroup();
  map.addLayer(cellsLayer);
  map.addLayer(proximityRadiiLayer);
  map.addLayer(cluster);
  map.addLayer(campsiteLayer);
  map.addLayer(routesLayer);
  map.addLayer(outlineLayer);

  map.on("moveend", () => {
    persistMapView();
    showCachedViewport();
    if (!campsiteEdit.value) scheduleLoad();
  });

  if (enabled.value.powerspot && !canLoadPowerspots()) {
    enabled.value = { ...enabled.value, powerspot: false };
  }

  await loadPois();
  maybeShowTutorial();
});

onBeforeUnmount(() => {
  stopPlaceTool();
  if (campsiteNoteTimer) clearTimeout(campsiteNoteTimer);
  map?.remove();
});

watch(
  [
    enabled,
    baseLayerId,
    showCells,
    exportMode,
    showAllRoutes,
    showInactivePowerspots,
    groupByLayer,
    radiusOverlays,
  ],
  () => persistUiSettings(),
  { deep: true },
);
watch(campsitePois, () => {
  syncMarkers();
}, { deep: true });
watch(outline, () => {
  if (placeTool.value === "outline") return;
  renderOutline();
}, { deep: true });
watch(enabled, () => {
  syncMarkers();
  syncRouteOverlays();
}, { deep: true });
watch(showInactivePowerspots, () => {
  syncMarkers();
});
watch(radiusOverlays, () => {
  syncProximityRadii();
}, { deep: true });
watch(selected, () => {
  syncMarkers();
  refreshMarkerSelection();
});
watch(baseLayerId, (id) => applyBaseLayer(id));
watch(cartoApiKey, () => applyBaseLayer(baseLayerId.value));
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
  if (on) {
    if (sheetTab.value === "explore") sheetTab.value = "export";
  } else {
    exitCampsiteEdit();
    sheetTab.value = "explore";
  }
  nextTick(() => {
    map?.invalidateSize();
  });
});
watch(settingsOpen, (open) => {
  if (!open && !canLoadPowerspots()) {
    pendingPowerspotEnable = false;
  }
});

function onSheetSnap() {
  nextTick(() => map?.invalidateSize());
}

function openExportSheet() {
  exportMode.value = true;
  sheetTab.value = "export";
}

function enterCampsiteEdit() {
  // Finish in-progress area drawing (commits polygon + selects POIs inside).
  if (placeTool.value) setPlaceTool(null);
  exportMode.value = true;
  campsiteEdit.value = true;
  sheetTab.value = "export";
  // Keep selected POIs in cache; while locked only those (+ planned) are shown.
  for (const p of exportPois.value) {
    poiCache.set(p.id, p);
  }
  refreshCampsiteConflicts();
  syncMarkers();
  refreshMarkerSelection();
  nextTick(() => map?.invalidateSize());
}

function exitCampsiteEdit() {
  const was = campsiteEdit.value;
  campsiteEdit.value = false;
  setPlaceTool(null);
  proximityWarning.value = "";
  clearWarnCircle();
  if (exportMode.value) sheetTab.value = "export";
  else sheetTab.value = "explore";
  syncMarkers();
  if (was) scheduleLoad();
}

/** Commits draft area if it has ≥3 points. Returns whether a new area was saved. */
function commitOutlineDraft() {
  if (outlineDraft.value.length >= 3) {
    outline.value = outlineDraft.value.map((p) => [p[0], p[1]] as [number, number]);
    outlineDraft.value = [];
    selectPoisInsideOutline(outline.value);
    return true;
  }
  outlineDraft.value = [];
  return false;
}

/** POIs currently rendered on the map and allowed by layer toggles. */
function currentlyShownPois(): Poi[] {
  const byId = new Map<string, Poi>();
  for (const p of pois.value) {
    if (!isLayerEnabled(p)) continue;
    byId.set(p.id, p);
  }
  for (const id of markers.keys()) {
    if (byId.has(id)) continue;
    const p =
      campsitePois.value.find((c) => c.id === id) ??
      poiCache.get(id) ??
      exportPoisById.value.get(id);
    if (!p || p.source === "campsite") continue;
    if (!isLayerEnabled(p)) continue;
    byId.set(id, p);
  }
  return [...byId.values()];
}

/** Add currently shown POIs inside the drawn area to the export selection. */
function selectPoisInsideOutline(ring: [number, number][]) {
  if (ring.length < 3) return;
  if (!exportMode.value) exportMode.value = true;
  const next = new Set(selected.value);
  const byId = new Map(exportPoisById.value);
  for (const p of currentlyShownPois()) {
    if (p.type === "route") continue;
    if (!pointInPolygon(p.lat, p.lng, ring)) continue;
    next.add(p.id);
    byId.set(p.id, p);
  }
  exportPoisById.value = byId;
  selected.value = next;
}

/** Deselect POIs named like "… Campsite - …". */
function excludeCampsiteLabeledPois() {
  const next = new Set(selected.value);
  const byId = new Map(exportPoisById.value);
  for (const id of [...next]) {
    const p = byId.get(id) ?? poiCache.get(id);
    if (!p || !isCampsiteLabeledPoi(p)) continue;
    next.delete(id);
    byId.delete(id);
  }
  exportPoisById.value = byId;
  selected.value = next;
}

/**
 * Convert selected “… Campsite - …” POIs into planned pins (draggable / deletable).
 * Drops them from the existing selection so they no longer count as neighbors.
 */
function convertCampsiteLabeledToPlanned() {
  const nextSelected = new Set(selected.value);
  const byId = new Map(exportPoisById.value);
  const added: Poi[] = [];
  const have = new Set(campsitePois.value.map((p) => p.id));
  for (const id of [...nextSelected]) {
    const p = byId.get(id) ?? poiCache.get(id);
    if (!p || !isCampsiteLabeledPoi(p)) continue;
    nextSelected.delete(id);
    byId.delete(id);
    if (have.has(p.id)) continue;
    added.push(
      normalizePoi({
        ...p,
        source: "campsite",
        name: p.name,
      }),
    );
    have.add(p.id);
  }
  if (!added.length && nextSelected.size === selected.value.size) return;
  exportPoisById.value = byId;
  selected.value = nextSelected;
  // Drop old (non-draggable) markers so syncMarkers recreates them as planned pins.
  for (const p of added) {
    const m = markers.get(p.id);
    if (!m) continue;
    removeMarkerFromHost(m);
    markers.delete(p.id);
  }
  // Lock first so recreated markers are born draggable.
  if (!campsiteEdit.value) {
    exportMode.value = true;
    campsiteEdit.value = true;
    sheetTab.value = "export";
    for (const p of exportPois.value) poiCache.set(p.id, p);
  }
  if (added.length) {
    campsitePois.value = [...campsitePois.value, ...added];
    refreshCampsiteConflicts();
  }
  syncMarkers();
  refreshMarkerSelection();
  nextTick(() => map?.invalidateSize());
}

function setPlaceTool(tool: PlaceTool) {
  if (placeTool.value === "outline" && tool !== "outline") {
    commitOutlineDraft();
  }
  stopPlaceTool();
  placeTool.value = tool;
  proximityWarning.value = "";
  clearWarnCircle();
  if (!tool || !map) {
    renderOutline();
    return;
  }
  if (tool === "outline") {
    startOutlineDraw();
    return;
  }
  renderOutline();
  startPlacePoi(tool);
}

function stopPlaceTool() {
  stopPlaceSession?.();
  stopPlaceSession = null;
  if (placeTool.value === "outline" && outlineDraft.value.length) {
    // keep draft until finished or cleared
  }
  if (map) map.getContainer().style.cursor = "";
}

function newCampsiteId() {
  return `camp-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

/** Existing selected export POIs + other planned campsite POIs (not the whole viewport cache). */
function proximityPeers(excludeId?: string): Poi[] {
  const byId = new Map<string, Poi>();
  for (const p of exportPois.value) {
    if (p.type === "route") continue;
    byId.set(p.id, p);
  }
  for (const p of campsitePois.value) {
    byId.set(p.id, p);
  }
  return [...byId.values()].filter((p) => p.id !== excludeId);
}

function refreshCampsiteConflicts() {
  const next = new Set<string>();
  let warn = "";
  for (const p of campsitePois.value) {
    const near = findNearbyPoi(p.lat, p.lng, proximityPeers(p.id));
    if (!near) continue;
    next.add(p.id);
    if (!warn) {
      warn = `Within ${WAYFARER_MIN_SEPARATION_M}m of “${near.poi.name}” (${Math.round(near.meters)}m)`;
    }
  }
  campsiteConflicts.value = next;
  proximityWarning.value = warn;
  if (!next.size) clearWarnCircle();
}

function flashCampsiteNote(msg: string) {
  campsiteNote.value = msg;
  if (campsiteNoteTimer) clearTimeout(campsiteNoteTimer);
  campsiteNoteTimer = setTimeout(() => {
    campsiteNote.value = "";
    campsiteNoteTimer = null;
  }, 7000);
}

function startPlacePoi(type: "pokestop" | "gym" | "powerspot") {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L) return;
  map.getContainer().style.cursor = "crosshair";

  const onClick = (e: LeafletMouseEvent) => {
    L.DomEvent.stopPropagation(e);
    const labels = { pokestop: "Planned PokéStop", gym: "Planned Gym", powerspot: "Planned Powerspot" };
    const p: Poi = {
      id: newCampsiteId(),
      type,
      name: labels[type],
      icon: "",
      lat: e.latlng.lat,
      lng: e.latlng.lng,
      source: "campsite",
    };
    campsitePois.value = [...campsitePois.value, p];
    if (campsitePois.value.length === CAMPSITE_CA_SOFT_LIMIT) {
      flashCampsiteNote(
        `Community Ambassadors can usually submit up to ${CAMPSITE_CA_SOFT_LIMIT} POIs. You can keep placing for planning.`,
      );
    }
    refreshCampsiteConflicts();
    if (campsiteConflicts.value.has(p.id)) {
      showWarnCircle(p.lat, p.lng);
    } else {
      clearWarnCircle();
    }
  };

  map.on("click", onClick);
  stopPlaceSession = () => {
    map?.off("click", onClick);
  };
}

function startOutlineDraw() {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L) return;
  map.getContainer().style.cursor = "crosshair";
  outlineDraft.value = outline.value ? outline.value.map((p) => [p[0], p[1]] as [number, number]) : [];
  renderOutline();

  const finish = () => {
    commitOutlineDraft();
    placeTool.value = null;
    stopPlaceTool();
    renderOutline();
  };

  const onClick = (e: LeafletMouseEvent) => {
    const t = e.originalEvent.target as HTMLElement | null;
    if (t?.closest?.(OUTLINE_HANDLE_SEL)) return;
    L.DomEvent.stopPropagation(e);
    const pts = outlineDraft.value;
    if (pts.length >= 3) {
      const first = L.latLng(pts[0][0], pts[0][1]);
      if (first.distanceTo(e.latlng) < 20) {
        finish();
        return;
      }
    }
    outlineDraft.value = [...pts, [e.latlng.lat, e.latlng.lng]];
    renderOutline();
  };

  const onDblClick = (e: LeafletMouseEvent) => {
    const t = e.originalEvent.target as HTMLElement | null;
    if (t?.closest?.(OUTLINE_HANDLE_SEL)) return;
    L.DomEvent.stopPropagation(e);
    L.DomEvent.preventDefault(e.originalEvent);
    finish();
  };

  const onKeyDown = (ev: KeyboardEvent) => {
    if (ev.key === "Escape") {
      outlineDraft.value = [];
      placeTool.value = null;
      stopPlaceTool();
      renderOutline();
    } else if ((ev.key === "Backspace" || ev.key === "Delete") && outlineDraft.value.length) {
      ev.preventDefault();
      outlineDraft.value = outlineDraft.value.slice(0, -1);
      renderOutline();
    }
  };

  map.on("click", onClick);
  map.on("dblclick", onDblClick);
  document.addEventListener("keydown", onKeyDown);
  const wasDbl = map.doubleClickZoom.enabled();
  if (wasDbl) map.doubleClickZoom.disable();

  stopPlaceSession = () => {
    map?.off("click", onClick);
    map?.off("dblclick", onDblClick);
    document.removeEventListener("keydown", onKeyDown);
    if (wasDbl) map?.doubleClickZoom.enable();
    if (map) map.getContainer().style.cursor = "";
  };
}

function syncOutlineShape(pts: [number, number][]) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!L || !outlineShape) return;
  const latlngs = pts.map(([lat, lng]) => L.latLng(lat, lng));
  const editing = placeTool.value === "outline";
  if (editing && pts.length >= 3) {
    outlineShape.setLatLngs([...latlngs, latlngs[0]]);
  } else {
    outlineShape.setLatLngs(latlngs);
  }
  if (outlineFill) outlineFill.setLatLngs(latlngs);
}

function renderOutline() {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!outlineLayer || !L || outlineDragging) return;
  outlineLayer.clearLayers();
  outlineShape = null;
  outlineFill = null;
  const editing = placeTool.value === "outline";
  const pts = editing ? outlineDraft.value : outline.value;
  if (!pts || pts.length === 0) return;
  const latlngs = pts.map(([lat, lng]) => L.latLng(lat, lng));
  const closed = !editing && pts.length >= 3;

  if (pts.length >= 2) {
    if (closed) {
      outlineShape = L.polygon(latlngs, {
        color: "#5b8cff",
        weight: 2,
        fillColor: "#5b8cff",
        fillOpacity: 0.12,
      });
      outlineLayer.addLayer(outlineShape);
    } else {
      const ring = editing && pts.length >= 3 ? [...latlngs, latlngs[0]] : latlngs;
      outlineShape = L.polyline(ring, {
        color: "#5b8cff",
        weight: 3,
        dashArray: editing ? "6 4" : undefined,
      });
      outlineLayer.addLayer(outlineShape);
      if (editing && pts.length >= 3) {
        outlineFill = L.polygon(latlngs, {
          color: "#5b8cff",
          weight: 0,
          fillColor: "#5b8cff",
          fillOpacity: 0.1,
          interactive: false,
        });
        outlineLayer.addLayer(outlineFill);
      }
    }
  }

  latlngs.forEach((ll, index) => {
    if (!editing) {
      outlineLayer!.addLayer(
        L.circleMarker(ll, { radius: 4, color: "#5b8cff", fillColor: "#fff", fillOpacity: 1, weight: 2 }),
      );
      return;
    }
    const vertex = L.marker(ll, {
      draggable: true,
      autoPan: true,
      zIndexOffset: 1200,
      icon: L.divIcon({
        className: "outline-vertex-wrap",
        html: `<div class="outline-vertex${index === 0 ? " outline-vertex-first" : ""}" title="Drag to move · double-click to remove"></div>`,
        iconSize: [16, 16],
        iconAnchor: [8, 8],
      }),
    });
    vertex.on("click", (ev) => L.DomEvent.stopPropagation(ev));
    vertex.on("dblclick", (ev) => {
      L.DomEvent.stopPropagation(ev);
      L.DomEvent.preventDefault(ev.originalEvent);
      outlineDraft.value = outlineDraft.value.filter((_, i) => i !== index);
      renderOutline();
    });
    vertex.on("dragstart", () => {
      outlineDragging = true;
    });
    vertex.on("drag", (ev) => {
      const pos = (ev.target as Marker).getLatLng();
      const next = outlineDraft.value.map((p, i) =>
        i === index ? ([pos.lat, pos.lng] as [number, number]) : p,
      );
      outlineDraft.value = next;
      syncOutlineShape(next);
    });
    vertex.on("dragend", () => {
      outlineDragging = false;
      renderOutline();
    });
    outlineLayer!.addLayer(vertex);
  });

  if (editing && pts.length >= 2) {
    addOutlineMidpointHandles(L, latlngs);
  }
}

const OUTLINE_HANDLE_SEL = ".outline-vertex, .outline-vertex-wrap, .outline-midpoint, .outline-midpoint-wrap";

/** Mid-edge handles: drag to insert a vertex and split that segment. */
function addOutlineMidpointHandles(
  L: typeof import("leaflet").default,
  latlngs: import("leaflet").LatLng[],
) {
  if (!outlineLayer) return;
  const n = latlngs.length;
  const edges: { a: number; b: number; insertAt: number }[] = [];
  for (let i = 0; i < n - 1; i++) {
    edges.push({ a: i, b: i + 1, insertAt: i + 1 });
  }
  // Closing edge when the ring is shown (≥3 points).
  if (n >= 3) {
    edges.push({ a: n - 1, b: 0, insertAt: n });
  }

  for (const edge of edges) {
    const a = latlngs[edge.a];
    const b = latlngs[edge.b];
    const mid = L.latLng((a.lat + b.lat) / 2, (a.lng + b.lng) / 2);
    let liveIndex: number | null = null;
    let suppressClick = false;

    const marker = L.marker(mid, {
      draggable: true,
      autoPan: true,
      zIndexOffset: 1100,
      icon: L.divIcon({
        className: "outline-midpoint-wrap",
        html: `<div class="outline-midpoint" title="Drag or click to split edge"></div>`,
        iconSize: [12, 12],
        iconAnchor: [6, 6],
      }),
    });

    marker.on("click", (ev) => {
      L.DomEvent.stopPropagation(ev);
      if (suppressClick) {
        suppressClick = false;
        return;
      }
      // Click inserts a vertex in place (no drag needed).
      const pos = marker.getLatLng();
      const next = [...outlineDraft.value];
      next.splice(edge.insertAt, 0, [pos.lat, pos.lng]);
      outlineDraft.value = next;
      renderOutline();
    });
    marker.on("dblclick", (ev) => {
      L.DomEvent.stopPropagation(ev);
      L.DomEvent.preventDefault(ev.originalEvent);
    });
    marker.on("dragstart", () => {
      suppressClick = true;
      outlineDragging = true;
      const pos = marker.getLatLng();
      const next = [...outlineDraft.value];
      next.splice(edge.insertAt, 0, [pos.lat, pos.lng]);
      outlineDraft.value = next;
      liveIndex = edge.insertAt;
      syncOutlineShape(next);
    });
    marker.on("drag", (ev) => {
      if (liveIndex == null) return;
      const pos = (ev.target as Marker).getLatLng();
      const next = outlineDraft.value.map((p, i) =>
        i === liveIndex ? ([pos.lat, pos.lng] as [number, number]) : p,
      );
      outlineDraft.value = next;
      syncOutlineShape(next);
    });
    marker.on("dragend", () => {
      outlineDragging = false;
      liveIndex = null;
      renderOutline();
    });
    outlineLayer.addLayer(marker);
  }
}

function clearOutline() {
  outline.value = null;
  outlineDraft.value = [];
  renderOutline();
}

function clearSelection() {
  if (!selected.value.size) return;
  selected.value = new Set();
  exportPoisById.value = new Map();
  refreshMarkerSelection();
  syncMarkers();
  nextTick(() => map?.invalidateSize());
}

function clearCampsitePois() {
  campsitePois.value = [];
  campsiteConflicts.value = new Set();
  proximityWarning.value = "";
  clearWarnCircle();
  syncMarkers();
}

function removeCampsitePoi(id: string) {
  campsitePois.value = campsitePois.value.filter((p) => p.id !== id);
  refreshCampsiteConflicts();
  syncMarkers();
}

function showWarnCircle(lat: number, lng: number) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!map || !L) return;
  clearWarnCircle();
  warnCircle = L.circleMarker([lat, lng], {
    radius: 18,
    color: "#e74c3c",
    weight: 2,
    fillColor: "#e74c3c",
    fillOpacity: 0.2,
  });
  warnCircle.addTo(map);
}

function clearWarnCircle() {
  if (warnCircle && map) {
    map.removeLayer(warnCircle);
    warnCircle = null;
  }
}

function selectedAndCampsitePois(): Map<string, Poi> {
  const source = new Map<string, Poi>();
  for (const p of exportPois.value) source.set(p.id, p);
  for (const p of campsitePois.value) source.set(p.id, p);
  return source;
}

function visibleMapPois(desired?: Map<string, Poi>): Map<string, Poi> {
  if (desired) return desired;
  const source = new Map<string, Poi>();
  for (const [id] of markers) {
    const p =
      campsitePois.value.find((c) => c.id === id) ??
      exportPoisById.value.get(id) ??
      poiCache.get(id) ??
      pois.value.find((poi) => poi.id === id);
    if (p) source.set(id, p);
  }
  return source;
}

/** POI separation uses selection; lure/interaction use POIs currently on the map. */
function radiusSourceFor(def: RadiusOverlayDef, desired?: Map<string, Poi>): Map<string, Poi> {
  if (def.id === "wayfarer") return selectedAndCampsitePois();
  return visibleMapPois(desired);
}

function syncProximityRadii(desired?: Map<string, Poi>) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (!proximityRadiiLayer || !L) return;
  proximityRadiiLayer.clearLayers();
  if (!anyRadiusOverlayEnabled(radiusOverlays.value)) return;

  const active = RADIUS_OVERLAYS.filter((d) => radiusOverlays.value[d.id]);
  // Draw largest first so smaller rings stay visible on top.
  const ordered = [...active].sort((a, b) => b.meters - a.meters);
  for (const def of ordered) {
    for (const p of radiusSourceFor(def, desired).values()) {
      if (!radiusOverlayApplies(def, p)) continue;
      L.circle([p.lat, p.lng], {
        radius: def.meters,
        color: def.color,
        weight: 1,
        opacity: 0.55,
        fillColor: def.color,
        fillOpacity: 0.06,
        interactive: false,
      }).addTo(proximityRadiiLayer);
    }
  }
}

function importPlan(plan: CampfirePlan) {
  exportMode.value = true;
  const byId = new Map<string, Poi>();
  const ids: string[] = [];
  for (const p of plan.existingPois) {
    const n = normalizePoi(p);
    byId.set(n.id, n);
    poiCache.set(n.id, n);
    ids.push(n.id);
  }
  exportPoisById.value = byId;
  selected.value = new Set(ids);
  campsitePois.value = plan.campsitePois.map((p) => normalizePoi({ ...p, source: "campsite" }));
  outline.value = plan.outline && plan.outline.length >= 3 ? plan.outline : null;
  outlineDraft.value = [];
  try {
    const settings: ExportSettings = {
      ...DEFAULT_EXPORT_SETTINGS,
      ...plan.exportSettings,
      mapName: plan.mapName,
      fileName: plan.fileName,
      includeTypes: {
        ...DEFAULT_EXPORT_SETTINGS.includeTypes,
        ...plan.exportSettings.includeTypes,
      },
      layers: normalizeExportLayers(plan.layers),
    };
    localStorage.setItem("campfire-export.exportSettings", JSON.stringify(settings));
  } catch {
    /* ignore */
  }
  enterCampsiteEdit();
  renderOutline();
  if (map && (ids.length || campsitePois.value.length || outline.value)) {
    const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
    if (L) {
      const pts: [number, number][] = [];
      for (const p of [...exportPois.value, ...campsitePois.value]) pts.push([p.lat, p.lng]);
      if (outline.value) pts.push(...outline.value);
      if (pts.length) {
        map.fitBounds(L.latLngBounds(pts.map(([lat, lng]) => L.latLng(lat, lng))), {
          padding: [48, 48],
          maxZoom: 18,
        });
      }
    }
  }
}

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
  baseTileLayer = L.tileLayer(withCartoApiKey(def.url, cartoApiKey.value), {
    attribution: def.attribution,
    maxZoom: def.maxZoom,
    subdomains: def.subdomains ?? "abc",
  });
  baseTileLayer.addTo(map);
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
  return (
    campsitePois.value.find((c) => c.id === p.id) ??
    poiCache.get(p.id) ??
    exportPoisById.value.get(p.id) ??
    p
  );
}

function attachExportSelect(layer: Layer, p: Poi) {
  attachExportGestures(layer, () => togglePoi(resolvePoi(p)));
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
  if (campsiteEdit.value) return;
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
  // Mutual exclusive: Campfire login disables Wayfarer.
  clearWayfarer();
  if (pendingPowerspotEnable) {
    pendingPowerspotEnable = false;
    enabled.value = { ...enabled.value, powerspot: true };
  }
  closeSettings();
  await loadPois();
}

function onTokenClear() {
  clearToken();
  if (!wayfarerReady.value) {
    disablePowerspotLayer();
  }
  scheduleLoad();
}

function onWayfarerSave(payload: { enabled: boolean; session: string; xsrfToken: string }) {
  if (!saveWayfarer(payload)) return;
  // Mutual exclusive: Wayfarer login clears Campfire token.
  clearToken();
  if (pendingPowerspotEnable && wayfarerReady.value) {
    pendingPowerspotEnable = false;
    enabled.value = { ...enabled.value, powerspot: true };
  }
  if (!payload.enabled && !token.value) {
    disablePowerspotLayer();
  }
  closeSettings();
  scheduleLoad();
}

function onWayfarerClear() {
  clearWayfarer();
  if (!token.value) {
    disablePowerspotLayer();
  }
  scheduleLoad();
}

function disablePowerspotLayer() {
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
  if (canLoadPowerspots()) return ALL_TYPES;
  return ALL_TYPES.filter((t) => t !== "powerspot");
}

async function loadPois() {
  if (!map) return;
  if (campsiteEdit.value) {
    loading.value = false;
    syncMarkers();
    return;
  }
  const seq = ++loadSeq;
  const b = map.getBounds();
  const bbox = [b.getSouth(), b.getWest(), b.getNorth(), b.getEast()].join(",");
  loading.value = true;
  error.value = "";
  try {
    const types = requestTypes().join(",");
    const res = await fetch(
      `${config.public.apiBase}/api/pois?bbox=${encodeURIComponent(bbox)}&types=${encodeURIComponent(types)}`,
      { headers: wayfarerHeaders(authHeaders()) },
    );
    if (!res.ok) {
      if (res.status === 401 || res.status === 403) {
        const msg = await res.text();
        if (/wayfarer/i.test(msg)) {
          invalidateWayfarer();
          enabled.value = { ...enabled.value, powerspot: false };
          openSettings();
          throw new Error("Wayfarer session rejected. Paste fresh SESSION and XSRF-TOKEN.");
        }
        invalidateToken();
        if (!wayfarerReady.value) {
          enabled.value = { ...enabled.value, powerspot: false };
        }
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
    const cur = resolvePoi(p);
    const selectedNow = !campsiteEdit.value && selected.value.has(id);
    return poiPopupHtml(cur, selectedNow, displayType(cur));
  }, POI_POPUP_OPTS);
  layer.on("popupopen", () => {
    openPopupId = id;
    bindPopupToggle(layer, resolvePoi(p));
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
  const exportSelected = !campsiteEdit.value && selected.value.has(p.id);
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

function removeMarkerFromHost(marker: Marker) {
  cluster?.removeLayer(marker);
  campsiteLayer?.removeLayer(marker);
}

function applyCampsiteDragStyle(marker: Marker, tooClose: boolean) {
  const el = marker.getElement()?.querySelector(".poi-marker");
  if (!el) return;
  el.classList.add("campsite-pin");
  el.classList.toggle("campsite-too-close", tooClose);
  const glyph = el.querySelector(".poi-glyph") as HTMLElement | null;
  if (glyph) glyph.style.background = tooClose ? "#e74c3c" : CAMPSITE_MARKER.color;
}

function syncMarkers() {
  const L = (window as unknown as { L: typeof import("leaflet").default }).L;
  if (!cluster || !campsiteLayer || !routesLayer || !L) return;
  const desired = new Map<string, Poi>();
  if (campsiteEdit.value) {
    for (const p of exportPois.value) {
      if (!isLayerEnabled(p)) continue;
      desired.set(p.id, p);
    }
  } else {
    for (const p of pois.value) {
      if (isLayerEnabled(p)) desired.set(p.id, p);
    }
    for (const [id, p] of exportPoisById.value) {
      if (!selected.value.has(id)) continue;
      if (!isLayerEnabled(p)) continue;
      if (!desired.has(id)) desired.set(id, p);
    }
  }
  for (const p of campsitePois.value) desired.set(p.id, p);

  for (const [id, marker] of markers) {
    if (desired.has(id)) continue;
    removeMarkerFromHost(marker);
    markers.delete(id);
    hideRouteOverlay(id);
    if (openPopupId === id) openPopupId = null;
  }

  for (const p of desired.values()) {
    const host = p.source === "campsite" ? campsiteLayer : cluster;
    const existing = markers.get(p.id);
    if (existing) {
      // Don't touch a marker mid-drag (setIcon / setLatLng would abort the gesture).
      if (draggingCampsiteId === p.id) continue;
      // Existing→planned conversion: recreate so drag handlers attach on a campsite layer pin.
      const markedCampsite = !!(existing as Marker & { _campfireCampsite?: boolean })._campfireCampsite;
      if (p.source === "campsite" && !markedCampsite) {
        removeMarkerFromHost(existing);
        markers.delete(p.id);
      } else {
        if (p.source === "campsite" && cluster.hasLayer(existing)) {
          cluster.removeLayer(existing);
          campsiteLayer.addLayer(existing);
        }
        const ll = existing.getLatLng();
        if (ll.lat !== p.lat || ll.lng !== p.lng) existing.setLatLng([p.lat, p.lng]);
        refreshMarker(existing, p);
        if (p.source === "campsite") {
          if (campsiteEdit.value) existing.dragging?.enable();
          else existing.dragging?.disable();
        }
        continue;
      }
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
      draggable: p.source === "campsite" && campsiteEdit.value,
      autoPan: false,
    });
    attachPoiPopup(marker, p);
    if (p.source === "campsite") {
      (marker as Marker & { _campfireCampsite?: boolean })._campfireCampsite = true;
      marker.on("dragstart", () => {
        draggingCampsiteId = p.id;
      });
      marker.on("drag", () => {
        const dragLl = marker.getLatLng();
        const near = findNearbyPoi(dragLl.lat, dragLl.lng, proximityPeers(p.id));
        applyCampsiteDragStyle(marker, !!near);
        if (near) {
          showWarnCircle(dragLl.lat, dragLl.lng);
          proximityWarning.value = `Within ${WAYFARER_MIN_SEPARATION_M}m of “${near.poi.name}” (${Math.round(near.meters)}m)`;
        } else {
          clearWarnCircle();
        }
      });
      marker.on("dragend", () => {
        draggingCampsiteId = null;
        const endLl = marker.getLatLng();
        campsitePois.value = campsitePois.value.map((c) =>
          c.id === p.id ? { ...c, lat: endLl.lat, lng: endLl.lng } : c,
        );
        refreshCampsiteConflicts();
        applyCampsiteDragStyle(marker, campsiteConflicts.value.has(p.id));
        if (campsiteConflicts.value.has(p.id)) showWarnCircle(endLl.lat, endLl.lng);
        else clearWarnCircle();
      });
    } else {
      attachExportSelect(marker, p);
    }
    markers.set(p.id, marker);
    host.addLayer(marker);
  }

  syncRouteOverlays(desired);
  syncProximityRadii(desired);
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

function markerShowsExportSelect(p: Poi) {
  // Once locked, selection is the export set — no purple outline needed.
  if (campsiteEdit.value) return false;
  return selected.value.has(p.id);
}

function refreshMarker(marker: Marker, p: Poi) {
  const L = (window as unknown as { L?: typeof import("leaflet").default }).L;
  if (L) {
    marker.setIcon(pinIcon(L, p));
  }
  const selectedNow = markerShowsExportSelect(p);
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
    const p =
      campsitePois.value.find((c) => c.id === id) ??
      poiCache.get(id) ??
      exportPoisById.value.get(id) ??
      pois.value.find((poi) => poi.id === id);
    if (!p) continue;
    const selectedNow = markerShowsExportSelect(p);
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

function pinIcon(L: typeof import("leaflet").default, p: Poi, conflictOverride?: boolean) {
  const type = displayType(p);
  const inactive = isInactivePowerspot(p);
  const campsite = p.source === "campsite";
  const tooClose =
    campsite && (conflictOverride ?? campsiteConflicts.value.has(p.id));
  const meta = TYPE_META[type];
  const color = tooClose
    ? "#e74c3c"
    : campsite
      ? CAMPSITE_MARKER.color
      : inactive
        ? INACTIVE_POWERSPOT.color
        : meta.color;
  const image = inactive ? INACTIVE_POWERSPOT.image : meta.image;
  const sel = markerShowsExportSelect(p) ? " selected-pin" : "";
  const inactiveClass = inactive ? " inactive-powerspot" : "";
  const campClass = campsite ? (tooClose ? " campsite-pin campsite-too-close" : " campsite-pin") : "";
  const inner =
    type === "super_mega_gym" && !campsite
      ? `<img src="${meta.image}" alt="">`
      : `<span class="poi-glyph" style="background:${color};-webkit-mask-image:url(${image});mask-image:url(${image})"></span>`;
  return L.divIcon({
    className: "poi-icon-wrap",
    html: `<div class="poi-marker${sel}${inactiveClass}${campClass}">${inner}</div>`,
    iconSize: [32, 32],
    iconAnchor: [16, 30],
  });
}
</script>
