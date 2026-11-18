<template>
  <aside
    class="bottom-sheet"
    :class="{ dragging, [`snap-${snap}`]: true }"
    :style="sheetStyle"
    role="complementary"
  >
    <div
      ref="handleRef"
      class="bottom-sheet-handle-wrap"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @dblclick.prevent="cycleSnap"
    >
      <div class="bottom-sheet-handle" aria-hidden="true" />
      <button type="button" class="bottom-sheet-expand-btn" :title="`Panel: ${snap}`" @click.stop="cycleSnap">
        {{ snapLabel }}
      </button>
    </div>
    <div class="bottom-sheet-body">
      <slot />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { noteSheetViewport, setPeekChromePx, useBottomSheet, type SheetSnap } from "~/composables/useBottomSheet";

const emit = defineEmits<{
  snap: [snap: SheetSnap];
}>();

const { snap, dragging, heightPx, cycleSnap, onPointerDown, onPointerMove, onPointerUp } = useBottomSheet((s) =>
  emit("snap", s),
);

const isMobileSheet = ref(false);

const sheetStyle = computed(() => {
  if (!isMobileSheet.value) return undefined;
  return { height: `${heightPx.value}px` };
});

const snapLabel = computed(() => {
  if (snap.value === "peek") return "▴";
  if (snap.value === "half") return "▴▴";
  return "▾";
});

const handleRef = ref<HTMLElement | null>(null);

function updateViewport() {
  isMobileSheet.value = window.matchMedia("(max-width: 860px)").matches;
  noteSheetViewport(window.innerHeight);
  measureChrome();
  syncSheetOffset();
}

function measureChrome() {
  const handle = handleRef.value;
  const tabs = handle?.parentElement?.querySelector(".sheet-tabs");
  if (!handle || !(tabs instanceof HTMLElement)) return;
  const border = 1;
  setPeekChromePx(handle.offsetHeight + tabs.offsetHeight + border);
}

function syncSheetOffset() {
  const h = isMobileSheet.value ? heightPx.value : 0;
  document.documentElement.style.setProperty("--sheet-h", `${h}px`);
}

watch(heightPx, syncSheetOffset);

onMounted(() => {
  updateViewport();
  window.addEventListener("resize", updateViewport);
  nextTick(measureChrome);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", updateViewport);
  document.documentElement.style.removeProperty("--sheet-h");
});
</script>
