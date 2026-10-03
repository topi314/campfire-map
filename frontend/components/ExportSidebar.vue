<template>
  <div class="sidebar-inner export-sidebar-inner">
    <div class="sidebar-header">
      <div class="sidebar-header-text">
        <h1>Export</h1>
        <p class="sub">Select POIs, place campsite stops, then download KMZ or save your work to reopen later.</p>
      </div>
      <div class="sidebar-header-actions">
        <button
          type="button"
          class="sidebar-icon-btn"
          title="Open work"
          aria-label="Open work"
          @click="triggerImport"
        >
          <UiIcon name="folder" />
        </button>
        <button
          type="button"
          class="sidebar-icon-btn"
          title="Save work"
          aria-label="Save work"
          @click="savePlan"
        >
          <UiIcon name="download" />
        </button>
        <button
          type="button"
          class="sidebar-settings btn-with-icon"
          title="Close export panel"
          @click="closeExport"
        >
          <UiIcon name="check" />
          Done
        </button>
      </div>
    </div>

    <div class="section">
      <h2>Selection</h2>
      <p class="counts">
        {{ selected.size }} selected · {{ campsitePois.length }} planned
        <span v-if="outline && outline.length >= 3"> · area</span>
        <template v-if="!campsiteEdit"> · {{ visiblePois.length }} in view</template>
      </p>
      <div class="btn-row btn-row-nowrap">
        <button
          type="button"
          class="btn-with-icon"
          :class="{ primary: placeTool === 'outline' }"
          @click="setTool(placeTool === 'outline' ? null : 'outline')"
        >
          <UiIcon name="polygon" />
          {{ placeTool === 'outline' ? "Drawing…" : "Draw area" }}
        </button>
        <button
          type="button"
          class="btn-with-icon"
          :disabled="!outline?.length"
          @click="emit('clear-outline')"
        >
          <UiIcon name="trash" />
          Clear
        </button>
        <button
          v-if="campsiteEdit"
          type="button"
          class="btn-with-icon"
          @click="emit('exit-campsite')"
        >
          <UiIcon name="unlock" />
          Unlock
        </button>
        <button
          v-else
          type="button"
          class="btn-with-icon"
          @click="emit('enter-campsite')"
        >
          <UiIcon name="lock" />
          Lock
        </button>
      </div>
      <div class="btn-row" style="margin-top: 6px">
        <button
          type="button"
          class="btn-with-icon"
          :disabled="!selected.size"
          @click="emit('clear-selection')"
        >
          <UiIcon name="clear" />
          Clear selection
        </button>
      </div>
      <div v-if="campsiteLabeledSelectedCount" class="btn-row" style="margin-top: 6px">
        <button type="button" class="btn-with-icon" @click="emit('exclude-campsite-pois')">
          <UiIcon name="ban" />
          Exclude campsite POIs ({{ campsiteLabeledSelectedCount }})
        </button>
        <button type="button" class="btn-with-icon" @click="emit('convert-campsite-pois')">
          <UiIcon name="convert" />
          Convert campsite POIs to planned
        </button>
      </div>
      <p v-if="placeTool === 'outline'" class="counts" style="margin-top: 6px">
        Click to add points, drag corners to move, drag or click a mid-edge handle to split a side. Double-click a corner to remove. Finish near the first point or double-click the map — POIs inside are selected. For “… Campsite - …” names: exclude from selection, or convert campsite POIs to planned so you can move them.
      </p>
      <p v-else-if="campsiteEdit" class="counts" style="margin-top: 6px">
        Map locked — only selected and planned POIs are shown; no new fetches.
      </p>
    </div>

    <div class="section">
      <h2>Place</h2>
      <div class="btn-row">
        <button
          type="button"
          class="btn-with-icon"
          :class="{ primary: placeTool === 'pokestop' }"
          @click="setTool(placeTool === 'pokestop' ? null : 'pokestop')"
        >
          <TypeIcon type="pokestop" />
          PokéStop
        </button>
        <button
          type="button"
          class="btn-with-icon"
          :class="{ primary: placeTool === 'gym' }"
          @click="setTool(placeTool === 'gym' ? null : 'gym')"
        >
          <TypeIcon type="gym" />
          Gym
        </button>
        <button
          type="button"
          class="btn-with-icon"
          :class="{ primary: placeTool === 'powerspot' }"
          @click="setTool(placeTool === 'powerspot' ? null : 'powerspot')"
        >
          <TypeIcon type="powerspot" />
          Powerspot
        </button>
      </div>
      <label class="sort-toggle" style="margin-top: 8px; display: inline-flex">
        <input
          :checked="radiusOverlays.wayfarer"
          type="checkbox"
          @change="onRadiusChange('wayfarer', ($event.target as HTMLInputElement).checked)"
        />
        Show POI separation (30 m)
      </label>
      <p v-if="placeTool && placeTool !== 'outline'" class="counts" style="margin-top: 6px">
        Tap the map to place. Too-close spots are marked in red (under 30 m).
      </p>
      <p v-if="proximityWarning" class="export-warning" style="margin-top: 8px">{{ proximityWarning }}</p>
      <p v-if="campsiteNote" class="export-note" style="margin-top: 8px">{{ campsiteNote }}</p>
    </div>

    <div class="section section-search">
      <input v-model="query" class="search" placeholder="Search export POIs" />
    </div>

    <div class="poi-lists">
      <div class="poi-list-section export-panel">
        <div class="export-panel-header">
          <div class="export-panel-heading">
            <h3 class="poi-list-heading">
              Export list
              <span class="counts">({{ listed.length }})</span>
            </h3>
            <label class="sort-toggle">
              <input v-model="groupByLayer" type="checkbox" />
              Group by layer
            </label>
          </div>
          <div class="export-panel-actions">
            <button
              type="button"
              class="primary export-panel-btn btn-with-icon"
              :disabled="!canExport"
              @click="exportOpen = true"
            >
              <UiIcon name="export" />
              Export
            </button>
          </div>
        </div>
        <div class="poi-list">
          <template v-if="groupByLayer">
            <template v-for="group in listedGroups" :key="group.key">
              <button
                type="button"
                class="poi-group-heading"
                :class="{ collapsed: collapsedGroups.has(group.key) }"
                :aria-expanded="!collapsedGroups.has(group.key)"
                @click="toggleGroup(group.key)"
              >
                <span class="poi-group-chevron" aria-hidden="true" />
                <TypeIcon :type="group.type" :campsite="group.key.startsWith('campsite_')" />
                {{ group.label }}
                <span class="counts">({{ group.pois.length }})</span>
              </button>
              <template v-if="!collapsedGroups.has(group.key)">
                <div
                  v-for="p in group.pois"
                  :key="p.id"
                  class="poi-item"
                  :class="{ selected: selected.has(p.id) || p.source === 'campsite' }"
                  @click="onListClick(p)"
                >
                  <TypeIcon
                    v-if="p.source === 'campsite'"
                    :type="group.type"
                    campsite
                  />
                  <img v-else-if="p.icon" :src="p.icon" :alt="p.name" />
                  <TypeIcon v-else :type="group.type" />
                  <div>
                    <div>{{ p.name }}</div>
                    <div class="meta">
                      {{ p.source === "campsite" ? "Planned" : TYPE_META[group.type].label }}
                      <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                      <span v-if="p.source !== 'campsite' && !inViewIds.has(p.id)"> · off map</span>
                    </div>
                  </div>
                  <button
                    v-if="p.source === 'campsite'"
                    type="button"
                    class="poi-remove"
                    title="Remove"
                    @click.stop="emit('remove-campsite-poi', p.id)"
                  >
                    ×
                  </button>
                </div>
              </template>
            </template>
          </template>
          <template v-else>
            <div
              v-for="p in listed"
              :key="p.id"
              class="poi-item"
              :class="{ selected: selected.has(p.id) || p.source === 'campsite' }"
              @click="onListClick(p)"
            >
              <TypeIcon
                v-if="p.source === 'campsite'"
                :type="campListType(p)"
                campsite
              />
              <img v-else-if="p.icon" :src="p.icon" :alt="p.name" />
              <TypeIcon v-else :type="displayTypeFor(p)" />
              <div>
                <div>{{ p.name }}</div>
                <div class="meta">
                  <template v-if="p.source === 'campsite'">Planned · </template>
                  {{ TYPE_META[displayTypeFor(p)].label }}
                  <span v-if="isInactivePowerspot(p)"> · Inactive</span>
                  <span v-if="p.source !== 'campsite' && !inViewIds.has(p.id)"> · off map</span>
                </div>
              </div>
              <button
                v-if="p.source === 'campsite'"
                type="button"
                class="poi-remove"
                title="Remove"
                @click.stop="emit('remove-campsite-poi', p.id)"
              >
                ×
              </button>
            </div>
          </template>
          <p v-if="!listed.length" class="poi-list-empty">
            Select POIs on the map, or place campsite stops above.
          </p>
        </div>
      </div>
    </div>

    <input ref="fileInput" type="file" accept="application/json,.json" class="hidden-file" @change="onImportFile" />

    <ExportModal
      :open="exportOpen"
      :export-pois="allExportPois"
      :outline="outline"
      :exporting="exporting"
      @close="exportOpen = false"
      @export="runExport"
    />
  </div>
</template>

<script setup lang="ts">
import { type RadiusOverlayId, type RadiusOverlayState } from "~/constants/map";
import {
  DEFAULT_EXPORT_SETTINGS,
  filterPoisForExport,
  normalizeExportLayers,
  sanitizeFileName,
  type ExportSettings,
  type PlaceTool,
} from "~/types/export";
import {
  buildCampfirePlan,
  downloadCampfirePlan,
  readCampfirePlanFile,
  type CampfirePlan,
} from "~/types/plan";
import {
  LAYER_TYPES,
  TYPE_META,
  isCampsiteLabeledPoi,
  isInactivePowerspot,
  poiDisplayType,
  poiLayerVisible,
  type Poi,
  type PoiType,
} from "~/types/poi";

const props = defineProps<{
  pois: Poi[];
  exportPois: Poi[];
  campsitePois: Poi[];
  selected: Set<string>;
  enabled: Record<PoiType, boolean>;
  showInactivePowerspots: boolean;
  campsiteEdit: boolean;
  placeTool: PlaceTool;
  outline: [number, number][] | null;
  proximityWarning: string;
  campsiteNote: string;
}>();

const emit = defineEmits<{
  toggle: [p: Poi];
  "enter-campsite": [];
  "exit-campsite": [];
  "set-place-tool": [tool: PlaceTool];
  "clear-outline": [];
  "clear-selection": [];
  "exclude-campsite-pois": [];
  "convert-campsite-pois": [];
  "remove-campsite-poi": [id: string];
  "focus-poi": [p: Poi];
  "import-plan": [plan: CampfirePlan];
}>();

const exportMode = defineModel<boolean>("exportMode", { required: true });
const groupByLayer = defineModel<boolean>("groupByLayer", { required: true });
const radiusOverlays = defineModel<RadiusOverlayState>("radiusOverlays", { required: true });

function onRadiusChange(id: RadiusOverlayId, checked: boolean) {
  radiusOverlays.value = { ...radiusOverlays.value, [id]: checked };
}

const query = ref("");
const exportOpen = ref(false);
const exporting = ref(false);
const collapsedGroups = ref<Set<string>>(new Set());
const fileInput = ref<HTMLInputElement | null>(null);
const config = useRuntimeConfig();
const { authHeaders, invalidateToken } = useSessionToken();

const visiblePois = computed(() =>
  props.pois.filter((p) =>
    poiLayerVisible(p, props.enabled, { showInactivePowerspots: props.showInactivePowerspots }),
  ),
);

const campsiteLabeledSelectedCount = computed(
  () => props.exportPois.filter((p) => isCampsiteLabeledPoi(p)).length,
);

const inViewIds = computed(() => new Set(props.pois.map((p) => p.id)));

const allExportPois = computed(() => [
  ...props.exportPois,
  // Always tag planned pins so the server picks campsite_* amber icons.
  ...props.campsitePois.map((p) => ({ ...p, source: "campsite" as const })),
]);

const canExport = computed(
  () => allExportPois.value.length > 0 || (props.outline != null && props.outline.length >= 3),
);

const listed = computed(() => {
  const q = query.value.trim().toLowerCase();
  return allExportPois.value.filter((p) => !q || p.name.toLowerCase().includes(q));
});

const listedGroups = computed(() => {
  type Group = { key: string; type: PoiType; label: string; pois: Poi[] };
  const campTypes: PoiType[] = ["gym", "pokestop", "powerspot"];
  const groups: Group[] = [];
  for (const type of LAYER_TYPES) {
    const pois = listed.value
      .filter((p) => p.source !== "campsite" && poiDisplayType(p, props.enabled.super_mega_gym) === type)
      .sort((a, b) => a.name.localeCompare(b.name));
    if (pois.length) groups.push({ key: type, type, label: TYPE_META[type].label, pois });
  }
  for (const type of campTypes) {
    const pois = listed.value
      .filter((p) => p.source === "campsite" && p.type === type)
      .sort((a, b) => a.name.localeCompare(b.name));
    if (pois.length) {
      groups.push({
        key: `campsite_${type}`,
        type,
        label: `Planned ${TYPE_META[type].label}`,
        pois,
      });
    }
  }
  return groups;
});

function displayTypeFor(p: Poi): PoiType {
  return poiDisplayType(p, props.enabled.super_mega_gym);
}

/** Planned pins never use the super-mega artwork — gym glyph + amber tint. */
function campListType(p: Poi): PoiType {
  const t = displayTypeFor(p);
  return t === "super_mega_gym" ? "gym" : t;
}

function toggleGroup(key: string) {
  const next = new Set(collapsedGroups.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  collapsedGroups.value = next;
}

function closeExport() {
  if (props.campsiteEdit) emit("exit-campsite");
  exportMode.value = false;
}

function setTool(tool: PlaceTool) {
  // Place tools lock the map; draw area does not (use Lock map separately).
  if (tool && tool !== "outline" && !props.campsiteEdit) emit("enter-campsite");
  emit("set-place-tool", tool);
}

function onListClick(p: Poi) {
  if (p.source === "campsite") {
    emit("focus-poi", p);
    return;
  }
  if (props.campsiteEdit) {
    emit("focus-poi", p);
    return;
  }
  emit("toggle", p);
}

function loadSavedSettings(): ExportSettings {
  const base: ExportSettings = {
    ...DEFAULT_EXPORT_SETTINGS,
    includeTypes: { ...DEFAULT_EXPORT_SETTINGS.includeTypes },
    layers: normalizeExportLayers(DEFAULT_EXPORT_SETTINGS.layers),
  };
  try {
    const raw = localStorage.getItem("campfire-export.exportSettings");
    if (!raw) return base;
    const saved = JSON.parse(raw) as Partial<ExportSettings>;
    return {
      ...base,
      ...saved,
      includeTypes: { ...base.includeTypes, ...saved.includeTypes },
      layers: normalizeExportLayers(saved.layers ?? base.layers),
    };
  } catch {
    return base;
  }
}

function savePlan() {
  const settings = loadSavedSettings();
  const plan = buildCampfirePlan({
    mapName: settings.mapName,
    fileName: settings.fileName,
    existingPois: props.exportPois,
    campsitePois: props.campsitePois,
    outline: props.outline,
    settings,
  });
  downloadCampfirePlan(plan, sanitizeFileName(settings.fileName));
}

function triggerImport() {
  fileInput.value?.click();
}

async function onImportFile(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  try {
    const plan = await readCampfirePlanFile(file);
    if (props.exportPois.length || props.campsitePois.length) {
      if (!confirm("Replace the current plan with this file?")) return;
    }
    emit("import-plan", plan);
  } catch (e) {
    alert(e instanceof Error ? e.message : "Failed to import plan");
  }
}

async function exportBlob(settings: ExportSettings) {
  const pois = filterPoisForExport(allExportPois.value, settings);
  const res = await fetch(`${config.public.apiBase}/api/export`, {
    method: "POST",
    headers: authHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({
      name: settings.mapName,
      format: settings.format,
      pois,
      outline: props.outline && props.outline.length >= 3 ? props.outline : null,
      layers: settings.layers.map(({ name, types }) => ({ name, types })),
    }),
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
