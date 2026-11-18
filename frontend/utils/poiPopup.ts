import {
  CAMPSITE_MARKER,
  INACTIVE_POWERSPOT,
  TYPE_META,
  isInactivePowerspot,
  type Poi,
  type PoiType,
} from "~/types/poi";

function escapeHtml(s: string) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeAttr(s: string) {
  return escapeHtml(s);
}

function iconBlock(p: Poi, display: PoiType) {
  const meta = TYPE_META[display];
  const campsite = p.source === "campsite";
  if (!campsite && p.icon) {
    return `<img class="poi-popup-photo" src="${escapeAttr(p.icon)}" alt="" />`;
  }
  if (!campsite && display === "super_mega_gym") {
    return `<img class="poi-popup-photo" src="${escapeAttr(meta.image)}" alt="" />`;
  }
  const glyphType = campsite && display === "super_mega_gym" ? "gym" : display;
  const glyphMeta = TYPE_META[glyphType];
  const color = campsite
    ? CAMPSITE_MARKER.color
    : isInactivePowerspot(p)
      ? INACTIVE_POWERSPOT.color
      : glyphMeta.color;
  const image = !campsite && isInactivePowerspot(p) ? INACTIVE_POWERSPOT.image : glyphMeta.image;
  return `<span class="poi-popup-glyph" style="background:${color};-webkit-mask-image:url(${escapeAttr(image)});mask-image:url(${escapeAttr(image)})"></span>`;
}

export function poiPopupHtml(p: Poi, isSelected: boolean, display: PoiType = p.type) {
  const meta = TYPE_META[display];
  const typeLabel =
    p.source === "campsite"
      ? `Planned ${meta.label}`
      : isInactivePowerspot(p)
        ? `${meta.label} · Inactive`
        : meta.label;
  const metaLines: string[] = [];
  if (p.type === "route" && p.path && p.path.length > 1) {
    metaLines.push(`${p.path.length} waypoints`);
  }
  metaLines.push(`${p.lat.toFixed(5)}, ${p.lng.toFixed(5)}`);

  return `
    <div class="poi-popup">
      <div class="poi-popup-header">
        ${iconBlock(p, display)}
        <div class="poi-popup-titles">
          <div class="poi-popup-name">${escapeHtml(p.name)}</div>
          <div class="poi-popup-type">${escapeHtml(typeLabel)}</div>
        </div>
      </div>
      <div class="poi-popup-meta">${metaLines.map((line) => escapeHtml(line)).join("<br>")}</div>
      ${
        p.source === "campsite"
          ? `<button type="button" class="poi-popup-btn poi-popup-btn-danger" data-poi-delete>Delete</button>`
          : `<button type="button" class="poi-popup-btn${isSelected ? " is-selected" : ""}" data-poi-toggle>
        ${isSelected ? "Remove from export" : "Select for export"}
      </button>`
      }
    </div>
  `;
}

export function areaPopupHtml(
  area: { name: string; color: string; ring: [number, number][] },
  poiCount: number,
  allInsideSelected: boolean,
) {
  const corners = `${area.ring.length} corners`;
  const inside = poiCount === 1 ? "1 POI inside" : `${poiCount} POIs inside`;
  const selectLabel = allInsideSelected && poiCount > 0 ? "Deselect POIs" : "Select POIs";
  return `
    <div class="poi-popup">
      <div class="poi-popup-header">
        <span class="poi-popup-swatch" style="background:${escapeAttr(area.color)}"></span>
        <div class="poi-popup-titles">
          <div class="poi-popup-name">${escapeHtml(area.name)}</div>
          <div class="poi-popup-type">Area</div>
        </div>
      </div>
      <div class="poi-popup-meta">${escapeHtml(corners)}<br>${escapeHtml(inside)}</div>
      <button type="button" class="poi-popup-btn" data-area-edit>Edit shape</button>
      <button type="button" class="poi-popup-btn${allInsideSelected && poiCount > 0 ? " is-selected" : ""}" data-area-select-pois${poiCount > 0 ? "" : " disabled"}>
        ${selectLabel}
      </button>
      <button type="button" class="poi-popup-btn poi-popup-btn-danger" data-area-delete>Delete area</button>
    </div>
  `;
}

export const POI_POPUP_OPTS = {
  className: "poi-popup-wrap",
  maxWidth: 300,
  minWidth: 220,
  autoPan: true,
  autoPanPadding: [48, 48] as [number, number],
  closeButton: true,
  closeOnClick: true,
};
