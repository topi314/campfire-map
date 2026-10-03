export type SheetSnap = "peek" | "half" | "expanded";

const STORAGE_KEY = "campfire-export.sheetSnap";

const SNAP_VH: Record<SheetSnap, number> = {
  peek: 0.14,
  half: 0.45,
  expanded: 0.85,
};

const ORDER: SheetSnap[] = ["peek", "half", "expanded"];

export function sheetHeightPx(snap: SheetSnap, viewportH = typeof window !== "undefined" ? window.innerHeight : 800) {
  return Math.round(viewportH * SNAP_VH[snap]);
}

export function readStoredSheetSnap(): SheetSnap {
  if (!import.meta.client) return "peek";
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === "peek" || raw === "half" || raw === "expanded") return raw;
  } catch {
    /* ignore */
  }
  return "peek";
}

export function writeStoredSheetSnap(snap: SheetSnap) {
  if (!import.meta.client) return;
  try {
    localStorage.setItem(STORAGE_KEY, snap);
  } catch {
    /* ignore */
  }
}

export function nextSheetSnap(current: SheetSnap): SheetSnap {
  const i = ORDER.indexOf(current);
  return ORDER[(i + 1) % ORDER.length];
}

export function nearestSheetSnap(heightPx: number, viewportH: number): SheetSnap {
  let best: SheetSnap = "peek";
  let bestDist = Infinity;
  for (const snap of ORDER) {
    const d = Math.abs(sheetHeightPx(snap, viewportH) - heightPx);
    if (d < bestDist) {
      bestDist = d;
      best = snap;
    }
  }
  return best;
}

export function useBottomSheet(onSnap?: (snap: SheetSnap) => void) {
  const snap = ref<SheetSnap>(readStoredSheetSnap());
  const dragHeight = ref<number | null>(null);
  const dragging = ref(false);

  const heightPx = computed(() => {
    if (dragHeight.value != null) return dragHeight.value;
    return sheetHeightPx(snap.value);
  });

  function setSnap(next: SheetSnap) {
    snap.value = next;
    dragHeight.value = null;
    writeStoredSheetSnap(next);
    onSnap?.(next);
  }

  function cycleSnap() {
    setSnap(nextSheetSnap(snap.value));
  }

  let startY = 0;
  let startH = 0;

  function onPointerDown(ev: PointerEvent) {
    const target = ev.currentTarget as HTMLElement;
    target.setPointerCapture?.(ev.pointerId);
    dragging.value = true;
    startY = ev.clientY;
    startH = heightPx.value;
    dragHeight.value = startH;
  }

  function onPointerMove(ev: PointerEvent) {
    if (!dragging.value) return;
    const vh = window.innerHeight;
    const delta = startY - ev.clientY;
    const min = sheetHeightPx("peek", vh);
    const max = sheetHeightPx("expanded", vh);
    dragHeight.value = Math.min(max, Math.max(min, startH + delta));
  }

  function onPointerUp() {
    if (!dragging.value) return;
    dragging.value = false;
    const h = dragHeight.value ?? heightPx.value;
    setSnap(nearestSheetSnap(h, window.innerHeight));
  }

  return {
    snap,
    dragging,
    heightPx,
    setSnap,
    cycleSnap,
    onPointerDown,
    onPointerMove,
    onPointerUp,
  };
}
