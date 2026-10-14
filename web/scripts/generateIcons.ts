// Regenerates every app icon PNG from native/assets/mark.svg (and the favicon fallbacks from web/public/favicon.svg), and the phone's lock glyphs.
// Run from web/: `bun scripts/generateIcons.ts`. Rasterises in headless Chromium through the playwright dev dependency.
// Only the tokens below carry color; they are the Mono Console background and foreground from native/src/global.css.
import { readFileSync } from "node:fs";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

import { chromium } from "playwright";

const root = resolve(import.meta.dirname, "../..");
const mark = readFileSync(resolve(root, "native/assets/mark.svg"), "utf8");
const favicon = readFileSync(resolve(root, "web/public/favicon.svg"), "utf8");

const dark = { background: "#050505", foreground: "#f5f5f5" };
const light = { background: "#ececec", foreground: "#0a0a0a" };

// The hub glyph's drawn box is 18.6 of its 24 units (its corner nodes reach 11.8 units from the center); coverage is that box as a share of the canvas.
const glyphShare = 18.6 / 24;

interface Icon {
  path: string;
  size: number;
  coverage: number;
  foreground: string;
  background?: string;
}

const phone = "native/assets";
const webPublic = "web/public";

const icons: Icon[] = [
  { path: `${phone}/icon.png`, size: 1024, coverage: 0.6, foreground: dark.foreground, background: dark.background },
  // Adaptive layers: the launcher crops to a 66% circle, so the corner nodes (a 0.49 box reaches 62% across) stay inside it.
  { path: `${phone}/adaptive-foreground.png`, size: 1024, coverage: 0.49, foreground: dark.foreground },
  { path: `${phone}/adaptive-monochrome.png`, size: 1024, coverage: 0.49, foreground: "#ffffff" },
  { path: `${phone}/notification-icon.png`, size: 96, coverage: 0.72, foreground: "#ffffff" },
  { path: `${phone}/splash-icon.png`, size: 1024, coverage: 0.6, foreground: light.foreground },
  { path: `${phone}/splash-icon-dark.png`, size: 1024, coverage: 0.6, foreground: dark.foreground },
  { path: `${webPublic}/apple-touch-icon.png`, size: 180, coverage: 0.6, foreground: dark.foreground, background: dark.background },
  { path: `${webPublic}/icon-192.png`, size: 192, coverage: 0.6, foreground: dark.foreground, background: dark.background },
  { path: `${webPublic}/icon-512.png`, size: 512, coverage: 0.6, foreground: dark.foreground, background: dark.background },
];

// The phone's markdown renderer takes only images inline, so a locked mention's lock is lucide's lock in each theme's muted-foreground.
const tokens = readFileSync(resolve(root, "native/src/global.css"), "utf8");
const [darkMuted, lightMuted] = [...tokens.matchAll(/--color-muted-foreground: ([^;]+);/g)].map((m) => m[1]);
const lockGlyph = '<rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>';
const locks = [
  { path: `${phone}/lock-dark.png`, color: darkMuted },
  { path: `${phone}/lock-light.png`, color: lightMuted },
];
const lockSize = 48;

const favicons = [
  { path: `${webPublic}/favicon-32.png`, size: 32 },
  { path: `${webPublic}/favicon-16.png`, size: 16 },
];

const glyphPage = ({ size, coverage, foreground, background }: Icon) => {
  const box = (size * coverage) / glyphShare;
  return `<body style="margin:0;background:${background ?? "transparent"}">
    <div style="width:${size}px;height:${size}px;display:grid;place-items:center;color:${foreground}">
      <div style="width:${box}px;height:${box}px">${mark.replace("<svg ", '<svg width="100%" height="100%" ')}</div>
    </div></body>`;
};

const lockPage = (color: string | undefined) =>
  `<body style="margin:0;background:transparent;color:${color}"><svg xmlns="http://www.w3.org/2000/svg" width="${lockSize}" height="${lockSize}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${lockGlyph}</svg></body>`;

const faviconPage = (size: number) =>
  `<body style="margin:0;background:transparent">${favicon.replace(/width="24" height="24"/, `width="${size}" height="${size}"`)}</body>`;

const browser = await chromium.launch();
const page = await browser.newPage();

const render = async (path: string, size: number, html: string, opaque: boolean) => {
  await page.setViewportSize({ width: size, height: size });
  await page.setContent(html);
  const png = await page.screenshot({ omitBackground: !opaque });
  await mkdir(resolve(root, path, ".."), { recursive: true });
  await writeFile(resolve(root, path), png);
};

for (const icon of icons) await render(icon.path, icon.size, glyphPage(icon), icon.background !== undefined);
for (const { path, size } of favicons) await render(path, size, faviconPage(size), false);
for (const { path, color } of locks) await render(path, lockSize, lockPage(color), false);

await browser.close();
