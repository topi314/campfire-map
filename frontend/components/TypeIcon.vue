<template>
  <span
    class="type-icon"
    :class="{ 'type-icon-glyph': useMask }"
    :style="useMask ? glyphStyle : undefined"
    :title="TYPE_META[type].label"
  >
    <img v-if="!useMask" :src="src" alt="" />
  </span>
</template>

<script setup lang="ts">
import { TYPE_META, type PoiType } from "~/types/poi";

const props = defineProps<{
  type: PoiType;
}>();

const src = computed(() => TYPE_META[props.type].image);
const useMask = computed(() => props.type !== "super_mega_gym");
const glyphStyle = computed(() => ({
  backgroundColor: TYPE_META[props.type].color,
  WebkitMaskImage: `url(${src.value})`,
  maskImage: `url(${src.value})`,
}));
</script>
