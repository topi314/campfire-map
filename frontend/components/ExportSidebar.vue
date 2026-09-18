<template>
  <aside class="sidebar export-sidebar">
    <div class="sidebar-header">
      <div class="sidebar-header-text">
        <h1>Export</h1>
        <p class="sub">Optional — select POIs and download a KMZ for Google My Maps.</p>
      </div>
      <button type="button" class="sidebar-settings" title="Close export panel" @click="exportMode = false">
        Done
      </button>
    </div>

    <div class="section">
      <h2>Selection</h2>
      <p class="counts">{{ selected.size }} selected · {{ visiblePois.length }} in view</p>
      <div class="btn-row">
        <button type="button" @click="selectVisible">Select visible</button>
        <button type="button" @click="clearSelection">Clear</button>
        <button type="button" :class="{ primary: drawing }" @click="emit('toggle-draw')">
          {{ drawing ? "Drawing…" : "Draw area" }}
        </button>
      </div>
    </div>

    <div class="section section-search">
      <input v-model="query" class="search" placeholder="Search selected names" />
    </div>

    <div class="poi-lists">
      <div class="poi-list-section export-panel">
        <div class="export-panel-header">
          <div class="export-panel-heading">
            <h3 class="poi-list-heading">
              Selected <span class="counts">({{ exportListed.length }})</span>
            </h3>
            <label class="sort-toggle">
              <input v-model="groupByLayer" type="checkbox" />
              Group by layer
            </label>
          </div>
          <button
            type="button"
            class="primary export-panel-btn"
            :disabled="!exportPois.length"
            @click="exportOpen = true"
          >
            Export…
          </button>
        </div>
        <div class="poi-list">
          <template v-if="groupByLayer">
            <template v-for="group in exportListedGroups" :key="group.type">
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
                  class="poi-item selected"
                  @click="emit('toggle', p)"
                >
                  <img v-if="p.icon" :src="p.icon" :alt="p.name" />
                  <TypeIcon v-else :type="group.type" />
                  <div>
                    <div>{{ p.name }}</div>
                    <div class="meta">
                      {{ TYPE_META[group.type].label }}
                      <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                      <span v-if="!inViewIds.has(p.id)"> · off map</span>
                    </div>
                  </div>
                </div>
              </template>
            </template>
          </template>
          <template v-else>
            <div
              v-for="p in exportListed"
              :key="p.id"
              class="poi-item selected"
              @click="emit('toggle', p)"
            >
              <img v-if="p.icon" :src="p.icon" :alt="p.name" />
              <TypeIcon v-else :type="poiDisplayType(p, enabled.super_mega_gym)" />
              <div>
                <div>{{ p.name }}</div>
                <div class="meta">
                  {{ TYPE_META[poiDisplayType(p, enabled.super_mega_gym)].label }}
                  <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                  <span v-if="!inViewIds.has(p.id)"> · off map</span>
                </div>
              </div>
            </div>
          </template>
          <p v-if="!exportListed.length" class="poi-list-empty">Select POIs on the map to add them here.</p>
        </div>
      </div>
    </div>

    <ExportModal
      :open="exportOpen"
      :export-pois="exportPois"
      :exporting="exporting"
      @close="exportOpen = false"
      @export="runExport"
    />
  </aside>
</template>

<script setup lang="ts">
import { filterPoisForExport, sanitizeFileName, type ExportSettings } from "~/types/export";
import {
  LAYER_TYPES,
  TYPE_META,
  isInactivePowerspot,
  poiDisplayType,
  poiLayerVisible,
  type Poi,
  type PoiType,
} from "~/types/poi";

const props = defineProps<{
  pois: Poi[];
  exportPois: Poi[];
  selected: Set<string>;
  drawing: boolean;
  enabled: Record<PoiType, boolean>;
  showInactivePowerspots: boolean;
}>();

const emit = defineEmits<{
  toggle: [p: Poi];
  "select-ids": [ids: string[]];
  "toggle-draw": [];
}>();

const exportMode = defineModel<boolean>("exportMode", { required: true });
const groupByLayer = defineModel<boolean>("groupByLayer", { required: true });

const query = ref("");
const exportOpen = ref(false);
const exporting = ref(false);
const collapsedGroups = ref<Set<PoiType>>(new Set());
const config = useRuntimeConfig();
const { authHeaders, invalidateToken } = useSessionToken();

const visiblePois = computed(() =>
  props.pois.filter((p) =>
    poiLayerVisible(p, props.enabled, { showInactivePowerspots: props.showInactivePowerspots }),
  ),
);

const inViewIds = computed(() => new Set(props.pois.map((p) => p.id)));

const exportListed = computed(() => {
  const q = query.value.trim().toLowerCase();
  return props.exportPois.filter((p) => !q || p.name.toLowerCase().includes(q));
});

const exportListedGroups = computed(() => {
  const buckets = new Map<PoiType, Poi[]>();
  for (const t of LAYER_TYPES) buckets.set(t, []);
  for (const p of exportListed.value) {
    const t = poiDisplayType(p, props.enabled.super_mega_gym);
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

function toggleGroup(type: PoiType) {
  const next = new Set(collapsedGroups.value);
  if (next.has(type)) next.delete(type);
  else next.add(type);
  collapsedGroups.value = next;
}

function selectVisible() {
  emit(
    "select-ids",
    visiblePois.value.map((p) => p.id),
  );
}

function clearSelection() {
  emit("select-ids", []);
}

async function exportBlob(settings: ExportSettings) {
  const pois = filterPoisForExport(props.exportPois, settings);
  const res = await fetch(`${config.public.apiBase}/api/export`, {
    method: "POST",
    headers: authHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({ name: settings.mapName, format: settings.format, pois }),
  });
  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      invalidateToken();
      throw new Error("Session token rejected. Paste a fresh Campfire token.");
    }
    throw new Error(await res.text());
  }
  return res.blob();
}

async function runExport(settings: ExportSettings) {
  exporting.value = true;
  try {
    const blob = await exportBlob(settings);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${sanitizeFileName(settings.fileName)}.${settings.format}`;
    a.click();
    URL.revokeObjectURL(url);
    exportOpen.value = false;
  } catch (e) {
    alert(e instanceof Error ? e.message : "Export failed");
  } finally {
    exporting.value = false;
  }
}
</script>
