import { TYPE_META, type Poi, type PoiType } from "~/types/poi";

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
  if (p.icon) {
    return `<img class="poi-popup-photo" src="${escapeAttr(p.icon)}" alt="" />`;
  }
  if (display === "super_mega_gym") {
    return `<img class="poi-popup-photo" src="${escapeAttr(meta.image)}" alt="" />`;
  }
  return `<span class="poi-popup-glyph" style="background:${meta.color};-webkit-mask-image:url(${escapeAttr(meta.image)});mask-image:url(${escapeAttr(meta.image)})"></span>`;
}

export function poiPopupHtml(p: Poi, isSelected: boolean, display: PoiType = p.type) {
  const meta = TYPE_META[display];
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
          <div class="poi-popup-type">${escapeHtml(meta.label)}</div>
        </div>
      </div>
      <div class="poi-popup-meta">${metaLines.map((line) => escapeHtml(line)).join("<br>")}</div>
      <button type="button" class="poi-popup-btn${isSelected ? " is-selected" : ""}" data-poi-toggle>
        ${isSelected ? "Deselect" : "Select for export"}
      </button>
    </div>
  `;
}

export const POI_POPUP_OPTS = {
  className: "poi-popup-wrap",
  maxWidth: 300,
  minWidth: 220,
  autoPan: true,
  autoPanPadding: [48, 48] as [number, number],
  closeOnClick: false,
};
