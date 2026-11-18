<template>
  <div
    v-if="open"
    class="token-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="export-title"
    @click.self="emit('close')"
  >
    <form class="token-card export-card" @submit.prevent="submit">
      <div class="modal-header">
        <h2 id="export-title">Export</h2>
        <button type="button" class="modal-close" title="Close" aria-label="Close" @click="emit('close')">
          <UiIcon name="cancel" />
        </button>
      </div>
      <p class="export-sub">
        {{ exportCount }} POI{{ exportCount === 1 ? "" : "s" }} in the export list
        <span v-if="areaCount > 0"> · {{ areaCount }} area{{ areaCount === 1 ? "" : "s" }}</span>
        <span v-if="routeCount > 0"> · {{ routeCount }} campsite route{{ routeCount === 1 ? "" : "s" }}</span>.
        <template v-if="draft.format === 'csv'">
          Download a spreadsheet of those POIs. Routes, areas, campsite routes, and range rings stay in KMZ and KML.
        </template>
        <template v-else>
          Import the file in
          <a href="https://www.google.com/maps/d/" target="_blank" rel="noreferrer">Google My Maps</a>
          or open it in Google Earth.
        </template>
      </p>

      <label class="token-label" for="export-map-name">Map name</label>
      <input
        id="export-map-name"
        v-model="draft.mapName"
        class="search"
        type="text"
        placeholder="Pokémon GO map"
        autocomplete="off"
      />

      <label class="token-label" for="export-file-name">File name</label>
      <div class="export-filename-row">
        <input
          id="export-file-name"
          v-model="draft.fileName"
          class="search"
          type="text"
          placeholder="pogo-map"
          autocomplete="off"
        />
        <span class="export-filename-suffix">.{{ draft.format }}</span>
      </div>

      <div class="export-format-row">
        <div class="token-label">Format</div>
        <label class="filter">
          <input v-model="draft.format" type="radio" value="kmz" />
          KMZ
          <span class="counts">(recommended · zipped · icons included)</span>
        </label>
        <label class="filter">
          <input v-model="draft.format" type="radio" value="kml" />
          KML
          <span class="counts">(single file · icons inlined)</span>
        </label>
        <label class="filter">
          <input v-model="draft.format" type="radio" value="csv" />
          CSV
          <span class="counts">(spreadsheet · POIs only)</span>
        </label>
      </div>

      <div v-if="draft.format === 'csv'" class="export-settings">
        <div class="token-label">POI types</div>
        <template v-for="item in typeOptions" :key="item.type">
          <template v-if="item.type !== 'route'">
            <label class="filter">
              <input v-model="draft.csvTypes[item.type]" type="checkbox" />
              <TypeIcon :type="item.type" />
              {{ item.label }}
              <span class="counts">({{ csvTypeCounts[item.type] || 0 }})</span>
            </label>
            <label v-if="item.type === 'powerspot'" class="filter filter-nested">
              <input
                v-model="draft.csvIncludeInactivePowerspots"
                type="checkbox"
                :disabled="!draft.csvTypes.powerspot"
              />
              Inactive powerspots
            </label>
          </template>
        </template>

        <div class="token-label csv-section-label">Include</div>
        <label class="filter">
          <input v-model="draft.csvScope" type="radio" value="all" />
          All POIs
        </label>
        <label class="filter">
          <input v-model="draft.csvScope" type="radio" value="planned" />
          Planned only
        </label>
        <label class="filter">
          <input v-model="draft.csvScope" type="radio" value="existing" />
          Existing only
        </label>

        <div class="token-label csv-section-label">Columns</div>
        <div v-for="(col, index) in draft.csvColumns" :key="col.id" class="csv-column-row">
          <label class="filter">
            <input v-model="col.enabled" type="checkbox" />
            {{ csvColumnLabel(col.id) }}
          </label>
          <div class="csv-column-actions">
            <button type="button" :disabled="index === 0" title="Move up" @click="moveColumn(index, -1)">↑</button>
            <button
              type="button"
              :disabled="index === draft.csvColumns.length - 1"
              title="Move down"
              @click="moveColumn(index, 1)"
            >
              ↓
            </button>
          </div>
        </div>
      </div>

      <div v-if="draft.format !== 'csv'" class="export-settings">
        <div class="token-label">Include types</div>
        <template v-for="item in typeOptions" :key="item.type">
          <label class="filter">
            <input v-model="draft.includeTypes[item.type]" type="checkbox" />
            <TypeIcon :type="item.type" />
            {{ item.label }}
            <span class="counts">({{ typeCounts[item.type] || 0 }})</span>
          </label>
          <label v-if="item.type === 'powerspot'" class="filter filter-nested">
            <input
              v-model="draft.includeInactivePowerspots"
              type="checkbox"
              :disabled="!draft.includeTypes.powerspot"
            />
            Inactive powerspots
          </label>
          <label v-if="item.type === 'route'" class="filter filter-nested">
            <input
              v-model="draft.includeRoutePaths"
              type="checkbox"
              :disabled="!draft.includeTypes.route"
            />
            Full route paths
            <span class="counts">(start point only when off)</span>
          </label>
        </template>
      </div>

      <LayerEditor
        v-if="draft.format !== 'csv'"
        v-model:layers="draft.layers"
        :include-types="draft.includeTypes"
      />

      <div v-if="draft.format !== 'csv'" class="export-settings export-icons-section">
        <div class="token-label">My Maps icons</div>
        <p class="counts export-icons-hint">
          Download PNG icons matching this map to upload as custom layer icons in My Maps.
        </p>
        <button
          type="button"
          class="btn-with-icon"
          :disabled="downloadingIcons || exporting"
          @click="downloadIcons"
        >
          <UiIcon name="zip" />
          {{ downloadingIcons ? "Preparing icons…" : "Download icon pack (ZIP)" }}
        </button>
      </div>

      <p v-if="draft.format === 'csv' && !csvColumnCount" class="export-warning">Choose at least one column.</p>
      <p v-else-if="nothingToExport" class="export-warning">No POIs match the current export settings.</p>
      <p v-else-if="draft.format !== 'csv' && myMapsWarning" class="export-warning">{{ myMapsWarning }}</p>
      <p v-else-if="draft.format === 'csv'" class="counts export-summary">
        {{ filteredCount }} POI{{ filteredCount === 1 ? "" : "s" }} will be exported.
      </p>
      <p v-else class="counts export-summary">
        {{ filteredCount }} POI{{ filteredCount === 1 ? "" : "s" }}
        <span v-if="areaCount > 0"> + {{ areaCount }} area{{ areaCount === 1 ? "" : "s" }}</span>
        <span v-if="routeCount > 0"> + {{ routeCount }} campsite route{{ routeCount === 1 ? "" : "s" }}</span>
        will be exported into {{ folderEstimate }} layer{{ folderEstimate === 1 ? "" : "s" }}.
      </p>

      <div class="btn-row" style="margin-top: 14px">
        <button
          type="submit"
          class="primary btn-with-icon"
          :disabled="exporting || nothingToExport || (draft.format === 'csv' && !csvColumnCount)"
        >
          <UiIcon name="download" />
          {{ exporting ? "Exporting…" : `Download ${draft.format.toUpperCase()}` }}
        </button>
        <button type="button" class="btn-with-icon" :disabled="exporting" @click="emit('close')">
          <UiIcon name="cancel" />
          Cancel
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { downloadMyMapsIconsZip } from "~/utils/myMapsIcons";
import {
  DEFAULT_EXPORT_SETTINGS,
  csvColumnLabel,
  defaultCsvColumns,
  defaultCsvTypes,
  estimateMyMapsFolders,
  exportTypeLabels,
  filterPoisForCsv,
  filterPoisForExport,
  normalizeCsvColumns,
  normalizeCsvScope,
  normalizeCsvTypes,
  normalizeExportFormat,
  moveCsvColumn,
  normalizeExportLayers,
  sanitizeFileName,
  selectedCsvColumns,
  type ExportSettings,
} from "~/types/export";
import { isGymPoi, isSuperMegaPoi, type Poi } from "~/types/poi";

const STORAGE_KEY = "campfire-export.exportSettings";

const props = defineProps<{
  open: boolean;
  exportPois: Poi[];
  areaCount: number;
  routeCount: number;
  exporting: boolean;
}>();

const emit = defineEmits<{
  close: [];
  export: [settings: ExportSettings];
}>();

const draft = ref<ExportSettings>({
  ...DEFAULT_EXPORT_SETTINGS,
  includeTypes: { ...DEFAULT_EXPORT_SETTINGS.includeTypes },
  layers: normalizeExportLayers(DEFAULT_EXPORT_SETTINGS.layers),
  csvColumns: defaultCsvColumns(),
  csvTypes: defaultCsvTypes(),
});
const downloadingIcons = ref(false);
const typeOptions = exportTypeLabels();

function countTypes(pois: Poi[]) {
  const c: Record<string, number> = {};
  for (const p of pois) {
    if (p.source === "campsite") {
      const key = p.type === "gym" || p.type === "super_mega_gym" ? "gym" : p.type;
      c[key] = (c[key] || 0) + 1;
      continue;
    }
    if (isGymPoi(p)) {
      c.gym = (c.gym || 0) + 1;
      if (isSuperMegaPoi(p)) c.super_mega_gym = (c.super_mega_gym || 0) + 1;
      continue;
    }
    c[p.type] = (c[p.type] || 0) + 1;
  }
  return c;
}

const typeCounts = computed(() => countTypes(props.exportPois));

const csvScopedPois = computed(() =>
  props.exportPois.filter((p) => {
    const planned = p.source === "campsite";
    if (draft.value.csvScope === "planned") return planned;
    if (draft.value.csvScope === "existing") return !planned;
    return true;
  }),
);

const csvTypeCounts = computed(() => countTypes(csvScopedPois.value));

const exportCount = computed(() => props.exportPois.length);
const areaCount = computed(() => props.areaCount);
const routeCount = computed(() => props.routeCount);
const filteredPois = computed(() =>
  draft.value.format === "csv"
    ? filterPoisForCsv(props.exportPois, draft.value)
    : filterPoisForExport(props.exportPois, draft.value),
);
const filteredCount = computed(() => filteredPois.value.length);

const folderEstimate = computed(() =>
  estimateMyMapsFolders(filteredPois.value, draft.value.layers, areaCount.value, routeCount.value),
);

const csvColumnCount = computed(() => selectedCsvColumns(draft.value.csvColumns).length);

const nothingToExport = computed(() =>
  draft.value.format === "csv"
    ? filteredCount.value === 0
    : filteredCount.value === 0 && areaCount.value === 0 && routeCount.value === 0,
);

const myMapsWarning = computed(() => {
  const folders = folderEstimate.value;
  if (folders > 10) {
    return `My Maps allows 10 layers; this export would create ${folders} folders. Merge layers or split the selection.`;
  }
  return "";
});

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    const saved = loadSavedSettings();
    draft.value = {
      ...DEFAULT_EXPORT_SETTINGS,
      ...saved,
      format: normalizeExportFormat(saved?.format),
      includeTypes: {
        ...DEFAULT_EXPORT_SETTINGS.includeTypes,
        ...saved?.includeTypes,
      },
      includeInactivePowerspots:
        typeof saved?.includeInactivePowerspots === "boolean"
          ? saved.includeInactivePowerspots
          : DEFAULT_EXPORT_SETTINGS.includeInactivePowerspots,
      layers: normalizeExportLayers(saved?.layers ?? DEFAULT_EXPORT_SETTINGS.layers),
      csvColumns: normalizeCsvColumns(saved?.csvColumns),
      csvTypes: normalizeCsvTypes(saved?.csvTypes),
      csvScope: normalizeCsvScope(saved?.csvScope),
      csvIncludeInactivePowerspots:
        typeof saved?.csvIncludeInactivePowerspots === "boolean"
          ? saved.csvIncludeInactivePowerspots
          : DEFAULT_EXPORT_SETTINGS.csvIncludeInactivePowerspots,
      csvIncludeRoutePaths:
        typeof saved?.csvIncludeRoutePaths === "boolean"
          ? saved.csvIncludeRoutePaths
          : DEFAULT_EXPORT_SETTINGS.csvIncludeRoutePaths,
    };
    if (!draft.value.mapName.trim()) {
      draft.value.mapName = DEFAULT_EXPORT_SETTINGS.mapName;
    }
    if (!draft.value.fileName.trim()) {
      draft.value.fileName = sanitizeFileName(draft.value.mapName);
    }
  },
);

function moveColumn(index: number, delta: number) {
  draft.value.csvColumns = moveCsvColumn(draft.value.csvColumns, index, delta);
}

function loadSavedSettings(): Partial<ExportSettings> | null {
  if (!import.meta.client) return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as Partial<ExportSettings>;
  } catch {
    return null;
  }
}

function saveSettings(settings: ExportSettings) {
  if (!import.meta.client) return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* ignore */
  }
}

function submit() {
  const mapName = draft.value.mapName.trim() || DEFAULT_EXPORT_SETTINGS.mapName;
  const fileName = sanitizeFileName(draft.value.fileName);
  const settings: ExportSettings = {
    ...draft.value,
    mapName,
    fileName,
    format: normalizeExportFormat(draft.value.format),
    includeTypes: { ...draft.value.includeTypes },
    layers: normalizeExportLayers(draft.value.layers),
    csvColumns: normalizeCsvColumns(draft.value.csvColumns),
    csvTypes: normalizeCsvTypes(draft.value.csvTypes),
    csvScope: normalizeCsvScope(draft.value.csvScope),
  };
  saveSettings(settings);
  emit("export", settings);
}

async function downloadIcons() {
  downloadingIcons.value = true;
  try {
    await downloadMyMapsIconsZip();
  } catch (e) {
    alert(e instanceof Error ? e.message : "Failed to prepare icons");
  } finally {
    downloadingIcons.value = false;
  }
}
</script>
