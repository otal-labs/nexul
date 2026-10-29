import type { GatewayNodeData, HostnameNodeData, ServiceNodeData, ServiceNodeExtras } from "@/models/Topology";

// Sizes worked out from the text before render, shared by cards and layout so pills meet their rows on frame one.

const MONO_12 = 7.2;
const MONO_11 = 6.6;
const SANS_14 = 8;
const LINE = 16;

export const LANE_GAP = 56;
export const GATEWAY_W = 360;
export const SERVICE_W = 288;
export const EXTERNAL_W = 240;
export const PILL_MAX_W = 320;

const GATEWAY_HEADER_H = 56;
// The container name's column beside the icon tile and the status.
const GATEWAY_NAME_W = 220;
const GATEWAY_INNER_W = GATEWAY_W - 24;
const ROW_GAP = 12;
const ROW_PAD = 12;

// The pill's globe, gaps, padding, link icon and 1px border; a pill that needs attention adds its status dot.
const PILL_CHROME = 70;
// Every canvas card's 1px border, top and bottom.
const CARD_BORDER = 2;
const PILL_DOT = 14;
// 3px padding and the 1px border, top and bottom.
const PILL_PAD = 8;

// Greedy first-fit over the places a line may break, the way the browser fills a line.
export const lineCount = (tokens: string[], maxChars: number): number => {
  let lines = 1;
  let used = 0;
  for (const token of tokens) {
    if (used > 0 && used + token.length > maxChars) {
      lines++;
      used = 0;
    }
    used += token.length;
  }
  return lines;
};

// A hostname breaks only after a dot; each label between dots stays whole.
export const hostnameTokens = (hostname: string): string[] => hostname.split(".").map((part, i, all) => (i < all.length - 1 ? `${part}.` : part));

// A service name breaks after a hyphen or a space, the browser's own breaks, and the port stays on the last piece.
const hyphenTokens = (text: string): string[] => text.split(/(?<=[- ])/);

const pillChrome = (data: HostnameNodeData) => PILL_CHROME + (data.tone ? PILL_DOT : 0);

// First-fit after the dots; a last line holding one label ("com") takes the label before it, so no widow.
export const hostnameLines = (data: HostnameNodeData): string[] => {
  const max = Math.floor((PILL_MAX_W - pillChrome(data)) / MONO_12);
  const lines: string[][] = [[]];
  for (const token of hostnameTokens(data.hostname)) {
    const line = lines.at(-1)!;
    if (line.length > 0 && line.join("").length + token.length > max) {
      lines.push([token]);
      continue;
    }
    line.push(token);
  }
  const last = lines.at(-1)!;
  const before = lines.at(-2);
  if (before && last.length === 1 && before.length > 1 && last[0]!.length + before.at(-1)!.length <= max) last.unshift(before.pop()!);
  return lines.map((l) => l.join(""));
};

export const pillWidth = (data: HostnameNodeData): number =>
  Math.ceil(Math.max(...hostnameLines(data).map((l) => l.length)) * MONO_12) + pillChrome(data);

export const pillHeight = (data: HostnameNodeData): number => LINE * hostnameLines(data).length + PILL_PAD;

export const routeAddress = (r: GatewayNodeData["routes"][number]) => (r.address ? `(${r.address}:${r.port})` : "");

const ROW_CHARS = Math.floor(GATEWAY_INNER_W / MONO_12);
// The arrow and its space: wrapped lines of a name hang under the name, not under the arrow.
const ARROW_CHARS = 2;

// The address drops to its own line before a name is split; only a name wider than the row breaks at hyphens.
export const routeLines = (r: GatewayNodeData["routes"][number]): { name: number; address: boolean } => {
  const label = `${r.service}:${r.port}`;
  const address = routeAddress(r).length;
  if (ARROW_CHARS + label.length + (address ? address + ROW_GAP / MONO_12 : 0) <= ROW_CHARS) return { name: 1, address: false };
  return { name: lineCount(hyphenTokens(label), ROW_CHARS - ARROW_CHARS), address: address > 0 };
};

// A row is as tall as its wrapped label, and never shorter than the pill that feeds it, so the two stay level.
export const routeRowHeights = (data: GatewayNodeData, pills: Map<string, HostnameNodeData> = new Map()): number[] =>
  data.routes.map((r) => {
    const pill = pills.get(r.id);
    const { name, address } = routeLines(r);
    return Math.max(LINE * (name + (address ? 1 : 0)) + ROW_PAD, pill ? pillHeight(pill) + 4 : 0);
  });

export const footerFacts = (data: GatewayNodeData): string[] => [data.target, ...data.networks].filter((f): f is string => !!f);

const FOOTER_CHARS = Math.floor(GATEWAY_INNER_W / MONO_11);

// The machine and networks on one line joined by "·" when they fit, else one fact per line.
export const footerOnOneLine = (data: GatewayNodeData): boolean => footerFacts(data).join(" · ").length <= FOOTER_CHARS;

const footerLines = (data: GatewayNodeData): number => {
  if (footerOnOneLine(data)) return 1;
  return footerFacts(data).reduce((sum, f) => sum + lineCount(hyphenTokens(f), FOOTER_CHARS), 0);
};

export const gatewayHeaderHeight = (data: GatewayNodeData): number =>
  GATEWAY_HEADER_H + LINE * (lineCount(hyphenTokens(data.name), Math.floor(GATEWAY_NAME_W / MONO_12)) - 1);

// The top of each row, measured from the card's outer edge: under its border, the header and the list's top hairline.
export const routeRowOffsets = (data: GatewayNodeData, heights: number[]): number[] => {
  let y = 1 + gatewayHeaderHeight(data) + 1;
  return heights.map((h) => {
    const top = y;
    y += h;
    return top;
  });
};

export const gatewayHeight = (data: GatewayNodeData, pills?: Map<string, HostnameNodeData>): number =>
  CARD_BORDER +
  gatewayHeaderHeight(data) +
  1 +
  (data.rowHeights ?? routeRowHeights(data, pills)).reduce((a, b) => a + b, 0) +
  1 +
  16 +
  LINE * footerLines(data);

export const serviceHeight = (data: ServiceNodeData & ServiceNodeExtras): number => {
  const nameLines = lineCount(hyphenTokens(data.name), Math.floor((SERVICE_W - 24) / SANS_14));
  return CARD_BORDER + 10 + 20 * nameLines + 4 + LINE + 8 + (data.target ? 1 + 12 + LINE : 0);
};

export const externalHeight = (name: string, url?: string): number => {
  const nameLines = lineCount(hyphenTokens(name), Math.floor((EXTERNAL_W - 48) / SANS_14));
  const urlLines = url ? Math.ceil((url.length * MONO_12) / (EXTERNAL_W - 48)) : 0;
  return CARD_BORDER + 24 + 20 * nameLines + LINE * urlLines + LINE;
};

// The box's microheader is mono 11px with 0.12em tracking, across the box less its 12px insets and the count.
const NETWORK_LABEL_CHARS = Math.floor((SERVICE_W + 48 - 24 - 24) / (MONO_11 + 1.32));

// Room above a network box's first card: its padding, then the microheader's lines.
export const networkBoxTop = (name: string): number =>
  24 + 24 + LINE * (lineCount(`network · ${name}`.split(/(?<=[-_ ])/), NETWORK_LABEL_CHARS) - 1);
