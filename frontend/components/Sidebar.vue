<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="sidebar-header-text">
        <h1>Pokémon GO map</h1>
        <p class="sub">Explore gyms, stops, routes, and more in the current view.</p>
      </div>
      <div class="sidebar-header-actions">
        <button
          type="button"
          class="sidebar-export"
          :class="{ primary: exportMode }"
          :title="exportMode ? 'Close export panel' : 'Export to Google My Maps'"
          @click="exportMode = !exportMode"
        >
          {{ exportMode ? "Exit" : "Export" }}
          <span class="counts">({{ selected.size }})</span>
        </button>
        <button type="button" class="sidebar-icon-btn" title="Help" aria-label="Help" @click="openTutorial()">
          ?
        </button>
        <button type="button" class="sidebar-icon-btn" title="Settings" aria-label="Settings" @click="openSettings()">
          <svg class="sidebar-gear" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M19.14 12.94c.04-.31.06-.63.06-.94s-.02-.63-.06-.94l2.03-1.58a.5.5 0 0 0 .12-.64l-1.92-3.32a.5.5 0 0 0-.6-.22l-2.39.96a7.07 7.07 0 0 0-1.63-.94l-.36-2.54a.5.5 0 0 0-.5-.42h-3.84a.5.5 0 0 0-.5.42l-.36 2.54c-.6.24-1.13.55-1.63.94l-2.39-.96a.5.5 0 0 0-.6.22L2.71 8.84a.5.5 0 0 0 .12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94L2.83 14.5a.5.5 0 0 0-.12.64l1.92 3.32c.14.24.43.34.68.22l2.39-.96c.5.39 1.04.7 1.63.94l.36 2.54c.05.24.26.42.5.42h3.84c.24 0 .45-.18.5-.42l.36-2.54c.6-.24 1.13-.55 1.63-.94l2.39.96c.25.12.54.02.68-.22l1.92-3.32a.5.5 0 0 0-.12-.64l-2.03-1.58ZM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2Z"
            />
          </svg>
        </button>
      </div>
    </div>

    <div class="section" :class="{ 'section-collapsed': collapsedSections.has('baseMap') }">
      <button
        type="button"
        class="section-toggle"
        :aria-expanded="!collapsedSections.has('baseMap')"
        @click="toggleSection('baseMap')"
      >
        <span class="section-chevron" aria-hidden="true" />
        Base map
      </button>
      <div v-show="!collapsedSections.has('baseMap')" class="section-body">
        <MapBaseLayer v-model="baseLayerId" />
      </div>
    </div>

    <div class="section" :class="{ 'section-collapsed': collapsedSections.has('layers') }">
      <button
        type="button"
        class="section-toggle"
        :aria-expanded="!collapsedSections.has('layers')"
        @click="toggleSection('layers')"
      >
        <span class="section-chevron" aria-hidden="true" />
        Layers
      </button>
      <div v-show="!collapsedSections.has('layers')" class="section-body">
        <template v-for="t in layerTypes" :key="t">
          <label class="filter">
            <input
              :checked="enabled[t]"
              type="checkbox"
              @change="onLayerChange(t, ($event.target as HTMLInputElement).checked)"
            />
            <TypeIcon :type="t" />
            {{ TYPE_META[t].label }}
            <span class="counts">({{ counts[t] || 0 }})</span>
            <span v-if="t === 'powerspot' && !canLoadPowerspots" class="counts layer-hint"> · needs auth</span>
          </label>
          <label v-if="t === 'powerspot'" class="filter filter-nested">
            <input
              v-model="showInactivePowerspots"
              type="checkbox"
              :disabled="!enabled.powerspot"
            />
            Inactive
            <span class="counts">({{ inactivePowerspotCount }})</span>
          </label>
          <label v-if="t === 'route'" class="filter filter-nested">
            <input v-model="showAllRoutes" type="checkbox" :disabled="!enabled.route" />
            Show all paths
          </label>
        </template>
        <label class="filter">
          <input v-model="showCells" type="checkbox" />
          <span class="cell-toggle-icon" aria-hidden="true" />
          S2 cells
          <span class="counts">({{ cellLabel }})</span>
        </label>
      </div>
    </div>

    <div class="poi-lists" :class="{ 'poi-lists-collapsed': collapsedSections.has('inView') }">
      <div class="poi-list-section">
        <div class="poi-list-heading-row">
          <button
            type="button"
            class="section-toggle section-toggle-inline"
            :aria-expanded="!collapsedSections.has('inView')"
            @click="toggleSection('inView')"
          >
            <span class="section-chevron" aria-hidden="true" />
            In view
            <span class="counts">({{ listedCount }})</span>
          </button>
          <label v-show="!collapsedSections.has('inView')" class="sort-toggle">
            <input v-model="groupByLayer" type="checkbox" />
            Group by layer
          </label>
        </div>
        <div v-show="!collapsedSections.has('inView')" class="section section-search">
          <input v-model="query" class="search" placeholder="Search names in view" />
        </div>
        <div v-show="!collapsedSections.has('inView')" class="poi-list">
          <template v-if="groupByLayer">
            <template v-for="group in listedGroups" :key="group.type">
              <button
                type="button"
                class="poi-group-heading"
                :class="{ collapsed: collapsedGroups.has(group.type) }"
                :aria-expanded="!collapsedGroups.has(group.type)"
                @click="toggleGroup(group.type)"
              >
                <span class="poi-group-chevron" aria-hidden="true" />
                <TypeIcon :type="group.type" />
                {{ TYPE_META[group.type].label }}
                <span class="counts">({{ group.pois.length }})</span>
              </button>
              <template v-if="!collapsedGroups.has(group.type)">
                <div
                  v-for="p in group.pois"
                  :key="p.id"
                  class="poi-item"
                  :class="{ selected: selected.has(p.id) }"
                  @mousedown="onPoiMouseDown"
                  @click="onPoiClick(p, $event)"
                  @touchstart.passive="onPoiTouchStart(p)"
                  @touchmove.passive="onPoiTouchMove(p)"
                  @touchend="onPoiTouchEnd(p)"
                  @touchcancel="onPoiTouchEnd(p)"
                >
                  <img v-if="p.icon" :src="p.icon" :alt="p.name" />
                  <TypeIcon v-else :type="group.type" />
                  <div>
                    <div>{{ p.name }}</div>
                    <div class="meta">
                      {{ TYPE_META[group.type].label }}
                      <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                    </div>
                  </div>
                </div>
              </template>
            </template>
          </template>
          <template v-else>
            <div
              v-for="p in listed"
              :key="p.id"
              class="poi-item"
              :class="{ selected: selected.has(p.id) }"
              @mousedown="onPoiMouseDown"
              @click="onPoiClick(p, $event)"
              @touchstart.passive="onPoiTouchStart(p)"
              @touchmove.passive="onPoiTouchMove(p)"
              @touchend="onPoiTouchEnd(p)"
              @touchcancel="onPoiTouchEnd(p)"
            >
              <img v-if="p.icon" :src="p.icon" :alt="p.name" />
              <TypeIcon v-else :type="poiDisplayType(p, enabled.super_mega_gym)" />
              <div>
                <div>{{ p.name }}</div>
                <div class="meta">
                  {{ TYPE_META[poiDisplayType(p, enabled.super_mega_gym)].label }}
                  <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                </div>
              </div>
            </div>
          </template>
          <p v-if="!listedCount" class="poi-list-empty">No POIs in the current view.</p>
        </div>
      </div>
    </div>

    <div class="status" :class="{ error: !!error }">{{ status }}</div>
  </aside>
</template>

<script setup lang="ts">
import {
  LAYER_TYPES,
  TYPE_META,
  isGymPoi,
  isInactivePowerspot,
  isSuperMegaPoi,
  normalizePoiType,
  poiDisplayType,
  poiLayerVisible,
  type Poi,
  type PoiType,
} from "~/types/poi";
import { LONG_PRESS_MS } from "~/utils/exportGesture";

type SidebarSection = "baseMap" | "layers" | "inView";

const props = defineProps<{
  pois: Poi[];
  selected: Set<string>;
  loading: boolean;
  error: string;
  l14Count: number;
  l17Count: number;
}>();

const emit = defineEmits<{
  toggle: [p: Poi, ev?: MouseEvent, exportSelect?: boolean];
  "request-powerspot-auth": [];
}>();

const { openSettings, token } = useSessionToken();
const { ready: wayfarerReady } = useWayfarerCredentials();
const { openTutorial } = useTutorial();

const enabled = defineModel<Record<PoiType, boolean>>("enabled", { required: true });
const baseLayerId = defineModel<string>("baseLayerId", { required: true });
const showCells = defineModel<boolean>("showCells", { required: true });
const exportMode = defineModel<boolean>("exportMode", { required: true });
const showAllRoutes = defineModel<boolean>("showAllRoutes", { required: true });
const showInactivePowerspots = defineModel<boolean>("showInactivePowerspots", { required: true });
const groupByLayer = defineModel<boolean>("groupByLayer", { required: true });

const canLoadPowerspots = computed(() => !!token.value || wayfarerReady.value);

const layerTypes = LAYER_TYPES;
const cellLabel = computed(() => {
  const parts: string[] = [];
  if (props.l14Count > 0) parts.push(`L14 · ${props.l14Count}`);
  if (props.l17Count > 0) parts.push(`L17 · ${props.l17Count}`);
  return parts.length ? parts.join(" · ") : "0";
});
const query = ref("");
const collapsedGroups = ref<Set<PoiType>>(new Set());
const collapsedSections = ref<Set<SidebarSection>>(new Set());
const longPressTimers = new Map<string, ReturnType<typeof setTimeout>>();
const longPressFired = new Set<string>();

function clearPoiLongPress(id: string) {
  const timer = longPressTimers.get(id);
  if (timer) clearTimeout(timer);
  longPressTimers.delete(id);
}

function onPoiTouchStart(p: Poi) {
  longPressFired.delete(p.id);
  clearPoiLongPress(p.id);
  longPressTimers.set(
    p.id,
    setTimeout(() => {
      longPressFired.add(p.id);
      emit("toggle", p, undefined, true);
    }, LONG_PRESS_MS),
  );
}

function onPoiTouchMove(p: Poi) {
  clearPoiLongPress(p.id);
}

function onPoiTouchEnd(p: Poi) {
  clearPoiLongPress(p.id);
}

function onPoiMouseDown(ev: MouseEvent) {
  // Shift+click starts a native text selection; block it for export multi-select.
  if (ev.shiftKey) ev.preventDefault();
}

function onPoiClick(p: Poi, ev: MouseEvent) {
  if (longPressFired.has(p.id)) {
    longPressFired.delete(p.id);
    return;
  }
  if (ev.shiftKey) {
    ev.preventDefault();
    window.getSelection()?.removeAllRanges();
  }
  emit("toggle", p, ev);
}

function toggleSection(section: SidebarSection) {
  const next = new Set(collapsedSections.value);
  if (next.has(section)) next.delete(section);
  else next.add(section);
  collapsedSections.value = next;
}

function toggleGroup(type: PoiType) {
  const next = new Set(collapsedGroups.value);
  if (next.has(type)) next.delete(type);
  else next.add(type);
  collapsedGroups.value = next;
}

const counts = computed(() => {
  const c: Record<string, number> = {};
  for (const p of props.pois) {
    if (isGymPoi(p)) {
      c.gym = (c.gym || 0) + 1;
      if (isSuperMegaPoi(p)) {
        c.super_mega_gym = (c.super_mega_gym || 0) + 1;
      }
      continue;
    }
    c[normalizePoiType(p.type)] = (c[normalizePoiType(p.type)] || 0) + 1;
  }
  return c;
});

const inactivePowerspotCount = computed(
  () => props.pois.filter((p) => isInactivePowerspot(p)).length,
);

const visiblePois = computed(() =>
  props.pois.filter((p) =>
    poiLayerVisible(p, enabled.value, { showInactivePowerspots: showInactivePowerspots.value }),
  ),
);

const filteredPois = computed(() => {
  const q = query.value.trim().toLowerCase();
  return visiblePois.value.filter((p) => !q || p.name.toLowerCase().includes(q)).slice(0, 400);
});

const listed = computed(() => filteredPois.value);

const listedGroups = computed(() => {
  const buckets = new Map<PoiType, Poi[]>();
  for (const t of LAYER_TYPES) buckets.set(t, []);
  for (const p of filteredPois.value) {
    const t = poiDisplayType(p, enabled.value.super_mega_gym);
    const list = buckets.get(t);
    if (list) list.push(p);
  }
  for (const list of buckets.values()) {
    list.sort((a, b) => a.name.localeCompare(b.name));
  }
  return LAYER_TYPES.filter((t) => (buckets.get(t)?.length ?? 0) > 0).map((type) => ({
    type,
    pois: buckets.get(type)!,
  }));
});

const listedCount = computed(() => filteredPois.value.length);

const status = computed(() => {
  if (props.error) return props.error;
  if (props.loading) return "Loading map data…";
  return `${props.pois.length} POIs loaded`;
});

function onLayerChange(t: PoiType, checked: boolean) {
  if (t === "powerspot" && checked && !canLoadPowerspots.value) {
    emit("request-powerspot-auth");
    return;
  }
  enabled.value = { ...enabled.value, [t]: checked };
  if (t === "route" && !checked) {
    showAllRoutes.value = false;
  }
  if (t === "powerspot" && !checked) {
    showInactivePowerspots.value = true;
  }
}
</script>
