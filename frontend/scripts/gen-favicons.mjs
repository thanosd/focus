#!/usr/bin/env node
// Regenerate the favicon set from favicon.svg. Run from the frontend/ directory:
//
//   node scripts/gen-favicons.mjs
//
// Sharp is pulled in transitively by Next.js, so no extra dep is needed.
// ICO output isn't supported by sharp, so we wrap the 16/32/48 PNGs into a
// multi-image .ico ourselves.

import { readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const HERE = dirname(fileURLToPath(import.meta.url));
const DIR = resolve(HERE, "..", "public", "favicons");
const SVG_PATH = resolve(DIR, "favicon.svg");

const PNG_SIZES = [
  { name: "favicon-16x16.png", size: 16 },
  { name: "favicon-32x32.png", size: 32 },
  { name: "apple-touch-icon.png", size: 180 },
  { name: "android-chrome-192x192.png", size: 192 },
  { name: "android-chrome-512x512.png", size: 512 },
];

const ICO_SIZES = [16, 32, 48];

function buildIco(images) {
  const header = Buffer.alloc(6);
  header.writeUInt16LE(0, 0);
  header.writeUInt16LE(1, 2);
  header.writeUInt16LE(images.length, 4);

  const entries = [];
  const payloads = [];
  let offset = 6 + images.length * 16;
  for (const { size, png } of images) {
    const entry = Buffer.alloc(16);
    entry.writeUInt8(size === 256 ? 0 : size, 0);
    entry.writeUInt8(size === 256 ? 0 : size, 1);
    entry.writeUInt8(0, 2);
    entry.writeUInt8(0, 3);
    entry.writeUInt16LE(1, 4);
    entry.writeUInt16LE(32, 6);
    entry.writeUInt32LE(png.length, 8);
    entry.writeUInt32LE(offset, 12);
    entries.push(entry);
    payloads.push(png);
    offset += png.length;
  }
  return Buffer.concat([header, ...entries, ...payloads]);
}

async function main() {
  const svg = await readFile(SVG_PATH);
  for (const { name, size } of PNG_SIZES) {
    const buf = await sharp(svg).resize(size, size).png().toBuffer();
    await writeFile(resolve(DIR, name), buf);
  }
  const icoImages = await Promise.all(
    ICO_SIZES.map(async (size) => ({
      size,
      png: await sharp(svg).resize(size, size).png().toBuffer(),
    })),
  );
  await writeFile(resolve(DIR, "favicon.ico"), buildIco(icoImages));
  console.log(`wrote favicons into ${DIR}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
