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
  closeButton: true,
  closeOnClick: false,
};
