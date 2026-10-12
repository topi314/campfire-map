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
          <UiIcon name="folderOpen" />
        </button>
        <button
          type="button"
          class="sidebar-icon-btn"
          title="Save work"
          aria-label="Save work"
          @click="savePlan"
        >
          <UiIcon name="save" />
        </button>
        <button
          type="button"
          class="sidebar-icon-btn"
          title="Clear selection"
          aria-label="Clear selection"
          :disabled="!selected.size"
          @click="emit('clear-selection')"
        >
          <UiIcon name="clear" />
        </button>
        <button
          type="button"
          class="primary sidebar-export btn-with-icon"
          :disabled="!canExport"
          title="Export to Google My Maps"
          @click="exportOpen = true"
        >
          <UiIcon name="export" />
          Export
        </button>
      </div>
    </div>

    <div class="section" :class="{ 'section-collapsed': collapsedSections.has('area') }">
      <button
        type="button"
        class="section-toggle"
        :aria-expanded="!collapsedSections.has('area')"
        @click="toggleSection('area')"
      >
        <span class="section-chevron" aria-hidden="true" />
        Areas
        <span class="counts">({{ areas.length }})</span>
      </button>
      <div v-show="!collapsedSections.has('area')" class="section-body">
        <div class="btn-row btn-row-nowrap">
          <button
            type="button"
            class="btn-with-icon"
            :class="{ primary: drawingNewArea }"
            @click="onDrawAreaClick"
          >
            <UiIcon :name="drawingNewArea ? 'check' : 'polygon'" />
            {{ drawingNewArea ? "Done" : "Add area" }}
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
        <p v-if="campsiteEdit" class="counts" style="margin-top: 6px">
          Map locked — only selected and planned POIs are shown; no new fetches.
        </p>
        <div v-if="areas.length" class="area-list">
          <div
            v-for="area in areas"
            :key="area.id"
            class="area-item"
            :class="{ active: editingAreaId === area.id }"
          >
            <div class="area-name-row">
              <input
                class="area-color"
                type="color"
                :value="area.color || '#5b8cff'"
                :aria-label="`Color for ${area.name}`"
                title="Area color"
                @input="onAreaColor(area.id, ($event.target as HTMLInputElement).value)"
                @click.stop
              />
              <input
                class="area-name"
                :value="area.name"
                :aria-label="`Rename ${area.name}`"
                @change="onRenameArea(area.id, ($event.target as HTMLInputElement).value)"
                @click.stop
              />
            </div>
            <div class="area-actions">
              <button type="button" class="btn-with-icon" title="Focus" @click="emit('focus-area', area.id)">
                Focus
              </button>
              <button
                type="button"
                class="btn-with-icon"
                :class="{ primary: editingAreaId === area.id }"
                :title="editingAreaId === area.id ? 'Finish editing' : 'Edit shape'"
                @click="editingAreaId === area.id ? emit('finish-area-edit') : emit('edit-area', area.id)"
              >
                {{ editingAreaId === area.id ? "Done" : "Edit" }}
              </button>
              <button
                type="button"
                class="btn-with-icon"
                title="Select POIs inside"
                @click="emit('select-pois-in-area', area.id)"
              >
                Select POIs
              </button>
              <button
                type="button"
                class="poi-remove"
                title="Delete area"
                @click.stop="emit('delete-area', area.id)"
              >
                ×
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="section" :class="{ 'section-collapsed': collapsedSections.has('place') }">
      <button
        type="button"
        class="section-toggle"
        :aria-expanded="!collapsedSections.has('place')"
        @click="toggleSection('place')"
      >
        <span class="section-chevron" aria-hidden="true" />
        Place
        <span class="counts">({{ campsitePois.length }} planned)</span>
      </button>
      <div v-show="!collapsedSections.has('place')" class="section-body">
        <div class="btn-row btn-row-nowrap">
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
        <template v-if="campsiteLabeledSelectedCount">
          <p class="counts" style="margin-top: 8px">
            Selected campsite POIs ({{ campsiteLabeledSelectedCount }})
          </p>
          <div class="btn-row btn-row-nowrap" style="margin-top: 6px">
            <button
              type="button"
              class="btn-with-icon"
              title="Remove those POIs from the export selection"
              @click="emit('exclude-campsite-pois')"
            >
              <UiIcon name="ban" />
              Deselect them
            </button>
            <button
              type="button"
              class="btn-with-icon"
              title="Turn those POIs into movable planned pins you can place"
              @click="emit('convert-campsite-pois')"
            >
              <UiIcon name="convert" />
              Make planned
            </button>
          </div>
        </template>
        <p v-if="placeTool && placeTool !== 'outline'" class="counts" style="margin-top: 6px">
          Tap the map to place. Too-close spots are marked in red (under 30 m). Turn on Min POI spacing in Ranges to show rings.
        </p>
        <p v-if="proximityWarning" class="export-warning" style="margin-top: 8px">{{ proximityWarning }}</p>
        <p v-if="campsiteNote" class="export-note" style="margin-top: 8px">{{ campsiteNote }}</p>
      </div>
    </div>

    <div class="poi-lists" :class="{ 'poi-lists-collapsed': collapsedSections.has('exportList') }">
      <div class="poi-list-section">
        <div class="poi-list-heading-row">
          <button
            type="button"
            class="section-toggle section-toggle-inline"
            :aria-expanded="!collapsedSections.has('exportList')"
            @click="toggleSection('exportList')"
          >
            <span class="section-chevron" aria-hidden="true" />
            Export list
            <span class="counts">({{ listed.length }})</span>
          </button>
          <label v-show="!collapsedSections.has('exportList')" class="sort-toggle">
            <input v-model="groupByLayer" type="checkbox" />
            Group by layer
          </label>
        </div>
        <div v-show="!collapsedSections.has('exportList')" class="section section-search">
          <input v-model="query" class="search" placeholder="Search export POIs" />
        </div>
        <div v-show="!collapsedSections.has('exportList')" class="poi-list">
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
      :area-count="exportableAreaCount"
      :exporting="exporting"
      @close="exportOpen = false"
      @export="runExport"
    />
  </div>
</template>

<script setup lang="ts">
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
  type MapArea,
} from "~/types/plan";
import {
  LAYER_TYPES,
  TYPE_META,
  isCampsiteLabeledPoi,
  isInactivePowerspot,
  poiDisplayType,
  type Poi,
  type PoiType,
} from "~/types/poi";

const props = defineProps<{
  pois: Poi[];
  exportPois: Poi[];
  campsitePois: Poi[];
  selected: Set<string>;
  enabled: Record<PoiType, boolean>;
  campsiteEdit: boolean;
  placeTool: PlaceTool;
  areas: MapArea[];
  editingAreaId: string | null;
  proximityWarning: string;
  campsiteNote: string;
}>();

const emit = defineEmits<{
  toggle: [p: Poi];
  "enter-campsite": [];
  "exit-campsite": [];
  "set-place-tool": [tool: PlaceTool];
  "edit-area": [id: string];
  "finish-area-edit": [];
  "rename-area": [id: string, name: string];
  "set-area-color": [id: string, color: string];
  "delete-area": [id: string];
  "focus-area": [id: string];
  "select-pois-in-area": [id: string];
  "clear-selection": [];
  "exclude-campsite-pois": [];
  "convert-campsite-pois": [];
  "remove-campsite-poi": [id: string];
  "focus-poi": [p: Poi];
  "import-plan": [plan: CampfirePlan];
}>();

const exportMode = defineModel<boolean>("exportMode", { required: true });
const groupByLayer = defineModel<boolean>("groupByLayer", { required: true });

const query = ref("");
const exportOpen = ref(false);
const exporting = ref(false);
type ExportSidebarSection = "area" | "place" | "exportList";
const collapsedGroups = ref<Set<string>>(new Set());
const collapsedSections = ref<Set<ExportSidebarSection>>(new Set());
const fileInput = ref<HTMLInputElement | null>(null);
const config = useRuntimeConfig();

const campsiteLabeledSelectedCount = computed(
  () => props.exportPois.filter((p) => isCampsiteLabeledPoi(p)).length,
);

const inViewIds = computed(() => new Set(props.pois.map((p) => p.id)));

const allExportPois = computed(() => [
  ...props.exportPois,
  // Always tag planned pins so the server picks campsite_* amber icons.
  ...props.campsitePois.map((p) => ({ ...p, source: "campsite" as const })),
]);

const exportableAreas = computed(() => props.areas.filter((a) => a.ring.length >= 3));
const exportableAreaCount = computed(() => exportableAreas.value.length);

const canExport = computed(
  () => allExportPois.value.length > 0 || exportableAreaCount.value > 0,
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

function toggleSection(section: ExportSidebarSection) {
  const next = new Set(collapsedSections.value);
  if (next.has(section)) next.delete(section);
  else next.add(section);
  collapsedSections.value = next;
}

function toggleGroup(key: string) {
  const next = new Set(collapsedGroups.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  collapsedGroups.value = next;
}

function setTool(tool: PlaceTool) {
  // Place tools lock the map; draw area does not (use Lock map separately).
  if (tool && tool !== "outline" && !props.campsiteEdit) emit("enter-campsite");
  emit("set-place-tool", tool);
}

const drawingNewArea = computed(
  () => props.placeTool === "outline" && !props.editingAreaId,
);

/** Finish a new area draw with Done; otherwise start a fresh draw. */
function onDrawAreaClick() {
  if (drawingNewArea.value) {
    emit("finish-area-edit");
    return;
  }
  setTool("outline");
}

function onRenameArea(id: string, name: string) {
  emit("rename-area", id, name);
}

function onAreaColor(id: string, color: string) {
  emit("set-area-color", id, color);
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
    areas: props.areas,
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
    if (props.exportPois.length || props.campsitePois.length || props.areas.length) {
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
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: settings.mapName,
      format: settings.format,
      pois,
      areas: exportableAreas.value.map(({ name, color, ring }) => ({ name, color, ring })),
      layers: settings.layers.map(({ name, types }) => ({ name, types })),
    }),
  });
  if (!res.ok) {
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
