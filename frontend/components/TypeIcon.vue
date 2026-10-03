<template>
  <span
    class="type-icon"
    :class="{ 'type-icon-glyph': useMask }"
    :style="useMask ? glyphStyle : undefined"
    :title="title"
  >
    <img v-if="!useMask" :src="src" alt="" />
  </span>
</template>

<script setup lang="ts">
import { CAMPSITE_MARKER, TYPE_META, type PoiType } from "~/types/poi";

const props = withDefaults(
  defineProps<{
    type: PoiType;
    /** When true, use amber campsite tint instead of the live POI color. */
    campsite?: boolean;
    color?: string;
  }>(),
  { campsite: false },
);

const src = computed(() => TYPE_META[props.type].image);
const useMask = computed(() => props.type !== "super_mega_gym" || props.campsite);
const title = computed(() =>
  props.campsite ? `Planned ${TYPE_META[props.type].label}` : TYPE_META[props.type].label,
);
const glyphStyle = computed(() => ({
  backgroundColor:
    props.color ?? (props.campsite ? CAMPSITE_MARKER.color : TYPE_META[props.type].color),
  WebkitMaskImage: `url(${src.value})`,
  maskImage: `url(${src.value})`,
}));
</script>
