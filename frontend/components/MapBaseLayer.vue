<template>
  <select v-model="model" class="search sidebar-basemap-select" aria-label="Base map">
    <option v-for="m in BASE_MAPS" :key="m.id" :value="m.id">{{ m.label }}</option>
  </select>
</template>

<script setup lang="ts">
import { BASE_MAPS } from "~/constants/baseMaps";

const model = defineModel<string>({ required: true });
</script>
