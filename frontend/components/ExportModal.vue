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
        <span v-if="outlineLen >= 3"> · area</span>.
        Import the file in
        <a href="https://www.google.com/maps/d/" target="_blank" rel="noreferrer">Google My Maps</a>
        or open it in Google Earth.
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
          <span class="counts">(zipped · icons included)</span>
        </label>
        <label class="filter">
          <input v-model="draft.format" type="radio" value="kml" />
          KML
          <span class="counts">(single file · icons inlined)</span>
        </label>
      </div>

      <div class="export-settings">
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

      <LayerEditor v-model:layers="draft.layers" :include-types="draft.includeTypes" />

      <div class="export-settings export-icons-section">
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

      <p v-if="filteredCount === 0 && outlineLen < 3" class="export-warning">No POIs match the current export settings.</p>
      <p v-else-if="myMapsWarning" class="export-warning">{{ myMapsWarning }}</p>
      <p v-else class="counts export-summary">
        {{ filteredCount }} POI{{ filteredCount === 1 ? "" : "s" }}
        <span v-if="outlineLen >= 3"> + area</span>
        will be exported into {{ folderEstimate }} layer{{ folderEstimate === 1 ? "" : "s" }}.
      </p>

      <div class="btn-row" style="margin-top: 14px">
        <button
          type="submit"
          class="primary btn-with-icon"
          :disabled="exporting || (filteredCount === 0 && outlineLen < 3)"
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
  estimateMyMapsFolders,
  exportTypeLabels,
  filterPoisForExport,
  normalizeExportLayers,
  sanitizeFileName,
  type ExportSettings,
} from "~/types/export";
import { isGymPoi, isSuperMegaPoi, type Poi } from "~/types/poi";

const STORAGE_KEY = "campfire-export.exportSettings";

const props = defineProps<{
  open: boolean;
  exportPois: Poi[];
  outline: [number, number][] | null;
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
});
const downloadingIcons = ref(false);
const typeOptions = exportTypeLabels();

const typeCounts = computed(() => {
  const c: Record<string, number> = {};
  for (const p of props.exportPois) {
    if (p.source === "campsite") {
      const key =
        p.type === "gym" || p.type === "super_mega_gym"
          ? "gym"
          : p.type;
      c[key] = (c[key] || 0) + 1;
      continue;
    }
    if (isGymPoi(p)) {
      c.gym = (c.gym || 0) + 1;
      if (isSuperMegaPoi(p)) {
        c.super_mega_gym = (c.super_mega_gym || 0) + 1;
      }
      continue;
    }
    c[p.type] = (c[p.type] || 0) + 1;
  }
  return c;
});

const exportCount = computed(() => props.exportPois.length);
const outlineLen = computed(() => props.outline?.length ?? 0);
const filteredPois = computed(() => filterPoisForExport(props.exportPois, draft.value));
const filteredCount = computed(() => filteredPois.value.length);

const folderEstimate = computed(() =>
  estimateMyMapsFolders(filteredPois.value, draft.value.layers, outlineLen.value),
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
      format: saved?.format === "kml" || saved?.format === "kmz" ? saved.format : DEFAULT_EXPORT_SETTINGS.format,
      includeTypes: {
        ...DEFAULT_EXPORT_SETTINGS.includeTypes,
        ...saved?.includeTypes,
      },
      includeInactivePowerspots:
        typeof saved?.includeInactivePowerspots === "boolean"
          ? saved.includeInactivePowerspots
          : DEFAULT_EXPORT_SETTINGS.includeInactivePowerspots,
      layers: normalizeExportLayers(saved?.layers ?? DEFAULT_EXPORT_SETTINGS.layers),
    };
    if (!draft.value.mapName.trim()) {
      draft.value.mapName = DEFAULT_EXPORT_SETTINGS.mapName;
    }
    if (!draft.value.fileName.trim()) {
      draft.value.fileName = sanitizeFileName(draft.value.mapName);
    }
  },
);

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
    format: draft.value.format === "kml" ? "kml" : "kmz",
    includeTypes: { ...draft.value.includeTypes },
    layers: normalizeExportLayers(draft.value.layers),
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
