<template>
  <div
    v-if="open"
    class="token-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="tutorial-title"
    @click.self="emit('close')"
  >
    <div class="token-card export-card tutorial-card">
      <div class="modal-header">
        <h2 id="tutorial-title">Quick start</h2>
        <button type="button" class="modal-close" title="Close" aria-label="Close" @click="emit('close')">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M18.3 5.71a1 1 0 0 0-1.41 0L12 10.59 7.11 5.7A1 1 0 0 0 5.7 7.11L10.59 12 5.7 16.89a1 1 0 1 0 1.41 1.41L12 13.41l4.89 4.89a1 1 0 0 0 1.41-1.41L13.41 12l4.89-4.89a1 1 0 0 0 0-1.4Z"
            />
          </svg>
        </button>
      </div>

      <ul class="tutorial-list">
        <li>Pan and zoom the map to load gyms, stops, routes, and more for the current view.</li>
        <li>Use the layer toggles to show or hide POI types. Search and the <strong>In view</strong> list help you find places — click a row to jump there.</li>
        <li>
          <strong>Session token</strong> (Settings) is optional — only needed for Powerspots.
          To get one: sign in at
          <a href="https://campfire.nianticlabs.com/" target="_blank" rel="noreferrer">Campfire</a>,
          open DevTools → <strong>Network</strong>, filter <kbd>graphql</kbd>, open a request, and copy the
          <strong>Authorization</strong> header (<code>Bearer …</code>). Paste it in Settings.
        </li>
        <li>
          <strong>Select for export:</strong> hold <kbd>Shift</kbd> and click a marker on the map, or <kbd>Shift</kbd>-click a row in <strong>In view</strong>.
          On touch devices, <strong>long-press</strong> instead of Shift.
        </li>
        <li>Open <strong>Export</strong> to review your selection, draw an area, or download a KMZ for Google My Maps.</li>
      </ul>

      <div class="btn-row tutorial-actions">
        <button type="button" class="primary" @click="emit('close')">Got it</button>
      </div>

      <AppFooter />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  close: [];
}>();
</script>
