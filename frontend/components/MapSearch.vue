<template>
  <div class="map-search" @keydown.escape="closeSuggestions">
    <div class="map-search-panel" :class="{ open: panelOpen }">
      <form class="map-search-form" @submit.prevent="runSearch">
        <input
          ref="inputEl"
          v-model="query"
          class="search map-search-input"
          type="search"
          role="combobox"
          :placeholder="placeholder"
          autocomplete="off"
          aria-label="Search for a place"
          aria-autocomplete="list"
          :aria-expanded="suggestionsOpen"
          aria-controls="map-search-suggestions"
          :aria-activedescendant="activeDescendant"
          @focus="onFocus"
          @keydown.down.prevent="moveActive(1)"
          @keydown.up.prevent="moveActive(-1)"
          @keydown.enter.prevent="onEnter"
        />
        <button type="submit" class="map-search-go" :disabled="searching || !query.trim()">
          {{ searching ? "…" : "Go" }}
        </button>
      </form>
      <ul
        v-if="suggestionsOpen"
        id="map-search-suggestions"
        class="map-search-results"
        role="listbox"
      >
        <li v-for="(r, i) in results" :key="r.id" role="option" :aria-selected="i === activeIndex">
          <button
            :id="`map-search-option-${i}`"
            type="button"
            :class="{ active: i === activeIndex }"
            @mousedown.prevent="pick(r)"
          >
            {{ r.label }}
          </button>
        </li>
      </ul>
      <p v-else-if="error" class="map-search-empty">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { DEFAULT_MAP_LABEL } from "~/constants/map";
import { searchPlaces, type PlaceResult } from "~/composables/usePlaceSearch";

const emit = defineEmits<{
  go: [place: PlaceResult];
}>();

const config = useRuntimeConfig();
const MIN_CHARS = 2;
const DEBOUNCE_MS = 450;

const placeholder = `Search — e.g. ${DEFAULT_MAP_LABEL}`;
const query = ref("");
const searching = ref(false);
const error = ref("");
const results = ref<PlaceResult[]>([]);
const activeIndex = ref(-1);
const inputEl = ref<HTMLInputElement | null>(null);

let debounceTimer: ReturnType<typeof setTimeout> | null = null;
let abortCtrl: AbortController | null = null;
let suggestSeq = 0;
let suppressSuggest = false;
let lastRequestAt = 0;

const suggestionsOpen = computed(() => results.value.length > 0);
const panelOpen = computed(() => suggestionsOpen.value || !!error.value);
const activeDescendant = computed(() =>
  activeIndex.value >= 0 ? `map-search-option-${activeIndex.value}` : undefined,
);

watch(query, () => {
  if (suppressSuggest) return;
  error.value = "";
  activeIndex.value = -1;
  scheduleSuggest();
});

onBeforeUnmount(() => {
  if (debounceTimer) clearTimeout(debounceTimer);
  abortCtrl?.abort();
});

function isAbortError(e: unknown) {
  return (
    (typeof e === "object" &&
      e !== null &&
      "name" in e &&
      (e as { name: string }).name === "AbortError") ||
    (e instanceof Error && /aborted/i.test(e.message))
  );
}

function scheduleSuggest() {
  if (debounceTimer) clearTimeout(debounceTimer);
  const q = query.value.trim();
  if (q.length < MIN_CHARS) {
    abortCtrl?.abort();
    results.value = [];
    searching.value = false;
    error.value = "";
    return;
  }
  // Nominatim (via our proxy) is rate-limited; wait at least ~1s between requests.
  const sinceLast = Date.now() - lastRequestAt;
  const wait = Math.max(DEBOUNCE_MS, 1100 - sinceLast);
  debounceTimer = setTimeout(() => {
    void fetchSuggestions(q);
  }, wait);
}

async function fetchSuggestions(q: string) {
  abortCtrl?.abort();
  abortCtrl = new AbortController();
  const seq = ++suggestSeq;
  searching.value = true;
  error.value = "";
  lastRequestAt = Date.now();
  try {
    const found = await searchPlaces(q, {
      signal: abortCtrl.signal,
      apiBase: String(config.public.apiBase || ""),
    });
    if (seq !== suggestSeq) return;
    results.value = found;
    if (found.length === 0) {
      error.value = "No places found.";
    }
  } catch (e) {
    if (isAbortError(e) || seq !== suggestSeq) return;
    // Keep prior suggestions on transient failures.
    if (results.value.length === 0) {
      error.value = "Search failed. Try again.";
    }
  } finally {
    if (seq === suggestSeq) searching.value = false;
  }
}

async function runSearch() {
  if (debounceTimer) clearTimeout(debounceTimer);
  const q = query.value.trim();
  if (!q) return;
  if (activeIndex.value >= 0 && results.value[activeIndex.value]) {
    pick(results.value[activeIndex.value]);
    return;
  }
  await fetchSuggestions(q);
  if (results.value.length === 1) {
    pick(results.value[0]);
  }
}

function onEnter() {
  if (activeIndex.value >= 0 && results.value[activeIndex.value]) {
    pick(results.value[activeIndex.value]);
    return;
  }
  void runSearch();
}

function moveActive(delta: number) {
  if (!results.value.length) return;
  const n = results.value.length;
  activeIndex.value = (activeIndex.value + delta + n) % n;
}

function onFocus() {
  if (results.value.length === 0 && query.value.trim().length >= MIN_CHARS) {
    scheduleSuggest();
  }
}

function closeSuggestions() {
  results.value = [];
  activeIndex.value = -1;
}

function pick(place: PlaceResult) {
  abortCtrl?.abort();
  if (debounceTimer) clearTimeout(debounceTimer);
  suppressSuggest = true;
  query.value = place.label;
  results.value = [];
  activeIndex.value = -1;
  error.value = "";
  emit("go", place);
  nextTick(() => {
    suppressSuggest = false;
  });
}
</script>
