import { zipSync } from "fflate";
import { ALL_TYPES, INACTIVE_POWERSPOT, ROUTE_COLORS, TYPE_META, type PoiType } from "~/types/poi";

const ICON_SIZE = 128;

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`Failed to load image: ${src}`));
    img.src = src;
  });
}

async function svgToPng(svgUrl: string, color?: string, flipX = false): Promise<Uint8Array> {
  const svgText = await fetch(svgUrl).then((r) => {
    if (!r.ok) throw new Error(`Failed to fetch ${svgUrl}`);
    return r.text();
  });
  const colored = color ? svgText.replace(/currentColor/g, color) : svgText;
  const blob = new Blob([colored], { type: "image/svg+xml" });
  const url = URL.createObjectURL(blob);
  try {
    const img = await loadImage(url);
    const canvas = document.createElement("canvas");
    canvas.width = ICON_SIZE;
    canvas.height = ICON_SIZE;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Canvas not available");
    if (flipX) {
      ctx.translate(ICON_SIZE, 0);
      ctx.scale(-1, 1);
    }
    ctx.drawImage(img, 0, 0, ICON_SIZE, ICON_SIZE);
    const out = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("PNG encode failed"))), "image/png");
    });
    return new Uint8Array(await out.arrayBuffer());
  } finally {
    URL.revokeObjectURL(url);
  }
}

async function rasterizeType(type: PoiType): Promise<Uint8Array> {
  const meta = TYPE_META[type];
  if (type === "super_mega_gym") {
    const img = await loadImage(meta.image);
    const canvas = document.createElement("canvas");
    canvas.width = ICON_SIZE;
    canvas.height = ICON_SIZE;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Canvas not available");
    ctx.drawImage(img, 0, 0, ICON_SIZE, ICON_SIZE);
    const out = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("PNG encode failed"))), "image/png");
    });
    return new Uint8Array(await out.arrayBuffer());
  }
  return svgToPng(meta.image, meta.color);
}

const README = `Pokémon GO map icons for Google My Maps

Use these PNG files as custom layer icons in My Maps (Style → More icons → Custom icons).

Files:
  gym.png            — Gyms
  super_mega_gym.png — Super Mega Gyms
  pokestop.png       — PokéStops
  powerspot.png      — Powerspot (active)
  powerspot-inactive.png — Inactive powerspot
  route.png          — Routes (start marker)
  route-end.png      — Route end point

Icons are 128×128 px. My Maps works best with icons up to 64×64 px on the map.
`;

export async function downloadMyMapsIconsZip() {
  const files: Record<string, Uint8Array> = {
    "README.txt": new TextEncoder().encode(README),
  };

  for (const type of ALL_TYPES) {
    files[`${type}.png`] = await rasterizeType(type);
  }
  files["powerspot-inactive.png"] = await svgToPng(INACTIVE_POWERSPOT.image, INACTIVE_POWERSPOT.color);
  files["route-end.png"] = await svgToPng(TYPE_META.route.image, ROUTE_COLORS.end, true);

  const zipped = zipSync(files);
  const blob = new Blob([zipped], { type: "application/zip" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "pogo-mymaps-icons.zip";
  a.click();
  URL.revokeObjectURL(url);
}
