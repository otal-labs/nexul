// Discord's <t:unix> and <t:unix:style> codes, which people paste in and every reader sees in their own zone.
export const DISCORD_TIME_SOURCE = "<t:(-?\\d{1,13})(?::([tTdDfFRsS]))?>";

export type DiscordTimeStyle = "t" | "T" | "d" | "D" | "f" | "F" | "R" | "s" | "S";

export interface DiscordTime {
  date: Date;
  style: DiscordTimeStyle | undefined;
  // The code as written, which Copy code hands back so a re-paste stays local for every reader.
  code: string;
}

// Seconds past the dates JavaScript can hold read as typed text.
export const readDiscordTime = (code: string, unix: string, style: string | undefined): DiscordTime | null => {
  const date = new Date(Number(unix) * 1000);
  if (Number.isNaN(date.getTime())) return null;
  return { date, style: style as DiscordTimeStyle | undefined, code };
};

export type TimedTextPart = { kind: "text"; text: string } | { kind: "time"; time: DiscordTime };

const TIMED_TEXT = new RegExp(`\`[^\`\\n]+\`|${DISCORD_TIME_SOURCE}`, "g");

const pushText = (parts: TimedTextPart[], text: string) => {
  if (text === "") return;
  const last = parts.at(-1);
  if (last?.kind === "text") {
    last.text += text;
    return;
  }
  parts.push({ kind: "text", text });
};

// An inline code span keeps its <t:…> as typed, so a code can be shown as the code.
export const splitDiscordTimes = (text: string): TimedTextPart[] => {
  const parts: TimedTextPart[] = [];
  let cursor = 0;
  for (const m of text.matchAll(TIMED_TEXT)) {
    const [raw, unix, style] = m;
    const time = unix === undefined ? null : readDiscordTime(raw, unix, style);
    pushText(parts, text.slice(cursor, m.index + (time ? 0 : raw.length)));
    if (time) parts.push({ kind: "time", time });
    cursor = m.index + raw.length;
  }
  pushText(parts, text.slice(cursor));
  return parts;
};

export const hasRelativeTime = (text: string): boolean =>
  splitDiscordTimes(text).some((part) => part.kind === "time" && part.time.style === "R");

const STYLES: Record<Exclude<DiscordTimeStyle, "R">, Intl.DateTimeFormatOptions> = {
  t: { timeStyle: "short" },
  T: { timeStyle: "medium" },
  d: { dateStyle: "short" },
  D: { dateStyle: "long" },
  f: { dateStyle: "long", timeStyle: "short" },
  F: { dateStyle: "full", timeStyle: "short" },
  s: { dateStyle: "short", timeStyle: "short" },
  S: { dateStyle: "short", timeStyle: "medium" },
};

const SECOND_MS = 1000;
const MINUTE_MS = 60 * SECOND_MS;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

const UNITS: [string, number][] = [
  ["year", 365 * DAY_MS],
  ["month", 30 * DAY_MS],
  ["day", DAY_MS],
  ["hour", HOUR_MS],
  ["minute", MINUTE_MS],
];

// Spelled out in English because the phone's Hermes has no Intl.RelativeTimeFormat, so both apps read the same.
const relative = (date: Date, now: number): string => {
  const diffMs = date.getTime() - now;
  const [unit, size] = UNITS.find(([, ms]) => Math.abs(diffMs) >= ms) ?? ["second", SECOND_MS];
  const n = Math.round(Math.abs(diffMs) / size);
  const span = `${n} ${unit}${n === 1 ? "" : "s"}`;
  return diffMs > 0 ? `in ${span}` : `${span} ago`;
};

// No style reads as Discord's default, f.
export const formatDiscordTime = (time: Pick<DiscordTime, "date" | "style">, now: number = Date.now()): string => {
  if (time.style === "R") return relative(time.date, now);
  return time.date.toLocaleString(undefined, STYLES[time.style ?? "f"]);
};

// What hovering or opening one shows: the whole date and the time with its zone.
export const fullDiscordTime = (date: Date): string => date.toLocaleString(undefined, { dateStyle: "full", timeStyle: "long" });

const icsStamp = (ms: number): string => new Date(ms).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "");

const icsText = (s: string): string => s.replace(/\\/g, "\\\\").replace(/;/g, "\\;").replace(/,/g, "\\,").replace(/\r?\n/g, "\\n");

const utf8Length = (ch: string): number => {
  const cp = ch.codePointAt(0) ?? 0;
  if (cp < 0x80) return 1;
  if (cp < 0x800) return 2;
  if (cp < 0x10000) return 3;
  return 4;
};

// RFC 5545 folds a content line past 75 octets onto continuation lines that start with a space.
const fold = (line: string): string => {
  const out: string[] = [];
  let current = "";
  let octets = 0;
  for (const ch of line) {
    const size = utf8Length(ch);
    if (octets + size > 75) {
      out.push(current);
      current = " ";
      octets = 1;
    }
    current += ch;
    octets += size;
  }
  out.push(current);
  return out.join("\r\n");
};

// A one-hour event at the instant, for any calendar app to import; nothing leaves the reader's machine.
export const discordTimeIcs = (date: Date, title: string, now: number = Date.now()): string => {
  const start = date.getTime();
  return [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//Nexul//Timestamp//EN",
    "BEGIN:VEVENT",
    `UID:${start}-${now}@nexul`,
    `DTSTAMP:${icsStamp(now)}`,
    `DTSTART:${icsStamp(start)}`,
    `DTEND:${icsStamp(start + HOUR_MS)}`,
    `SUMMARY:${icsText(title)}`,
    "END:VEVENT",
    "END:VCALENDAR",
  ]
    .map(fold)
    .join("\r\n")
    .concat("\r\n");
};
