import type { Layer, LeafletMouseEvent, Marker } from "leaflet";

export const LONG_PRESS_MS = 500;

export function attachLongPress(el: HTMLElement | SVGElement, onLongPress: () => void) {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let moved = false;

  const clear = () => {
    if (timer) clearTimeout(timer);
    timer = null;
  };

  const onStart = (ev: TouchEvent) => {
    moved = false;
    clear();
    timer = setTimeout(() => {
      if (!moved) {
        ev.preventDefault();
        onLongPress();
      }
    }, LONG_PRESS_MS);
  };

  const onMove = () => {
    moved = true;
    clear();
  };

  const onEnd = () => clear();

  el.addEventListener("touchstart", onStart, { passive: false });
  el.addEventListener("touchmove", onMove, { passive: true });
  el.addEventListener("touchend", onEnd);
  el.addEventListener("touchcancel", onEnd);
}

export function eventHasShift(ev: LeafletMouseEvent) {
  const oe = ev.originalEvent;
  return !!(oe && "shiftKey" in oe && (oe as MouseEvent).shiftKey);
}

function layerElement(layer: Layer): HTMLElement | null {
  if ("getElement" in layer && typeof layer.getElement === "function") {
    return (layer as Marker).getElement() ?? null;
  }
  return null;
}

export function attachExportGestures(
  layer: Layer,
  onExportSelect: () => void,
  opts?: { shiftClick?: boolean },
) {
  if (opts?.shiftClick !== false) {
    layer.on("click", (ev) => {
      if (!eventHasShift(ev as LeafletMouseEvent)) return;
      const Lany = (window as unknown as { L?: typeof import("leaflet").default }).L;
      Lany?.DomEvent.stop(ev as LeafletMouseEvent);
      onExportSelect();
      if ("closePopup" in layer && typeof layer.closePopup === "function") {
        layer.closePopup();
      }
    });
  }

  const bindLongPress = () => {
    const el = layerElement(layer);
    if (!el || el.dataset.exportGesture === "1") return;
    el.dataset.exportGesture = "1";
    attachLongPress(el, onExportSelect);
  };

  layer.on("add", bindLongPress);
  bindLongPress();
}
