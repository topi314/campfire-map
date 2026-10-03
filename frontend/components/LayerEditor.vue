<template>
  <div class="layer-editor">
    <div class="token-label">My Maps layers</div>
    <p class="counts layer-editor-hint">
      Assign POI types to named layers. Drag types between layers. Routes (start, path, end) stay in one layer.
    </p>
    <div class="btn-row layer-preset-row">
      <button type="button" class="btn-with-icon" @click="applyPreset('campsite')">
        <TypeIcon type="pokestop" campsite />
        Campsite layout
      </button>
      <button type="button" class="btn-with-icon" @click="applyPreset('website')">
        <TypeIcon type="gym" />
        By type
      </button>
    </div>

    <div v-for="(layer, index) in layers" :key="layer.id" class="layer-card">
      <div class="layer-card-header">
        <input v-model="layer.name" class="search layer-name-input" type="text" :aria-label="`Layer ${index + 1} name`" />
        <div class="layer-card-actions">
          <button type="button" :disabled="index === 0" title="Move up" @click="moveLayer(index, -1)">↑</button>
          <button type="button" :disabled="index === layers.length - 1" title="Move down" @click="moveLayer(index, 1)">
            ↓
          </button>
          <button type="button" :disabled="layers.length <= 1" title="Remove layer" @click="removeLayer(index)">
            ×
          </button>
        </div>
      </div>
      <div class="layer-chips" @dragover.prevent @drop="onDrop($event, layer.id)">
        <span
          v-for="t in visibleTypes(layer.types)"
          :key="t"
          class="layer-chip"
          draggable="true"
          @dragstart="onDragStart($event, t)"
        >
          {{ LAYER_TYPE_LABELS[t] }}
        </span>
        <span v-if="!visibleTypes(layer.types).length" class="counts">Drop types here</span>
      </div>
    </div>

    <div class="layer-card layer-unassigned" @dragover.prevent @drop="onDrop($event, null)">
      <div class="layer-card-header">
        <strong>Unassigned</strong>
        <span class="counts">(omitted from export)</span>
      </div>
      <div class="layer-chips">
        <span
          v-for="t in unassigned"
          :key="t"
          class="layer-chip muted"
          draggable="true"
          @dragstart="onDragStart($event, t)"
        >
          {{ LAYER_TYPE_LABELS[t] }}
        </span>
        <span v-if="!unassigned.length" class="counts">All included types assigned</span>
      </div>
    </div>

    <button type="button" class="layer-add-btn btn-with-icon" @click="addLayer">
      <UiIcon name="plus" />
      Add layer
    </button>
  </div>
</template>

<script setup lang="ts">
import {
  LAYER_TYPE_LABELS,
  campsiteLayerTypeIncluded,
  defaultCampsiteExportLayers,
  defaultWebsiteExportLayers,
  newLayerId,
  unassignedLayerTypes,
  type ExportLayer,
  type LayerTypeKey,
} from "~/types/export";
import type { PoiType } from "~/types/poi";

const props = defineProps<{
  includeTypes: Record<PoiType, boolean>;
}>();

const layers = defineModel<ExportLayer[]>("layers", { required: true });

const unassigned = computed(() => unassignedLayerTypes(layers.value, props.includeTypes));

let dragType: LayerTypeKey | null = null;

function visibleTypes(types: LayerTypeKey[]) {
  return types.filter((t) => campsiteLayerTypeIncluded(t, props.includeTypes));
}

function applyPreset(kind: "website" | "campsite") {
  layers.value =
    kind === "campsite" ? defaultCampsiteExportLayers() : defaultWebsiteExportLayers();
}

function onDragStart(ev: DragEvent, t: LayerTypeKey) {
  if (!campsiteLayerTypeIncluded(t, props.includeTypes)) {
    ev.preventDefault();
    return;
  }
  dragType = t;
  ev.dataTransfer?.setData("text/plain", t);
  if (ev.dataTransfer) ev.dataTransfer.effectAllowed = "move";
}

function onDrop(ev: DragEvent, targetLayerId: string | null) {
  ev.preventDefault();
  const t = (ev.dataTransfer?.getData("text/plain") as LayerTypeKey) || dragType;
  dragType = null;
  if (!t || !campsiteLayerTypeIncluded(t, props.includeTypes)) return;
  const next = layers.value.map((l) => ({
    ...l,
    types: l.types.filter((x) => x !== t),
  }));
  if (targetLayerId) {
    const target = next.find((l) => l.id === targetLayerId);
    if (target && !target.types.includes(t)) target.types.push(t);
  }
  layers.value = next;
}

function moveLayer(index: number, delta: number) {
  const next = [...layers.value];
  const j = index + delta;
  if (j < 0 || j >= next.length) return;
  const [item] = next.splice(index, 1);
  next.splice(j, 0, item);
  layers.value = next;
}

function removeLayer(index: number) {
  if (layers.value.length <= 1) return;
  layers.value = layers.value.filter((_, i) => i !== index);
}

function addLayer() {
  layers.value = [
    ...layers.value,
    { id: newLayerId(), name: `Layer ${layers.value.length + 1}`, types: [] },
  ];
}
</script>
