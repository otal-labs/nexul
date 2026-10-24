// @vitest-environment node
import { describe, expect, it } from "vitest";

import {
  discordTimeIcs,
  formatDiscordTime,
  hasRelativeTime,
  splitDiscordTimes,
  type DiscordTimeStyle,
} from "@nexul/client-core/discordTime";

// 2026-10-24T18:39:25Z: the seconds read 25 in every zone, and the year 2026.
const UNIX = 1792867165;
const date = new Date(UNIX * 1000);

describe("splitDiscordTimes", () => {
  it("reads every style and a bare code, keeping the code as written", () => {
    for (const style of ["t", "T", "d", "D", "f", "F", "R", "s", "S"]) {
      const code = `<t:${UNIX}:${style}>`;
      expect(splitDiscordTimes(`at ${code}.`)).toEqual([
        { kind: "text", text: "at " },
        { kind: "time", time: { date, style, code } },
        { kind: "text", text: "." },
      ]);
    }
    expect(splitDiscordTimes(`<t:${UNIX}>`)).toEqual([{ kind: "time", time: { date, style: undefined, code: `<t:${UNIX}>` } }]);
  });

  it("leaves an unknown style, a malformed code and an impossible date as typed", () => {
    for (const text of [`<t:${UNIX}:x>`, `<t:${UNIX}:>`, "<t:abc>", "<t:>", `<t: ${UNIX}>`, "<t:9999999999999>"]) {
      expect(splitDiscordTimes(text)).toEqual([{ kind: "text", text }]);
    }
  });

  it("leaves a code inside an inline code span as typed", () => {
    expect(splitDiscordTimes(`paste \`<t:${UNIX}:F>\` then <t:${UNIX}:R>`)).toEqual([
      { kind: "text", text: `paste \`<t:${UNIX}:F>\` then ` },
      { kind: "time", time: { date, style: "R", code: `<t:${UNIX}:R>` } },
    ]);
  });

  it("returns nothing for empty text", () => {
    expect(splitDiscordTimes("")).toEqual([]);
  });
});

describe("hasRelativeTime", () => {
  it("is true only for a relative code outside code spans", () => {
    expect(hasRelativeTime(`soon <t:${UNIX}:R>`)).toBe(true);
    expect(hasRelativeTime(`\`<t:${UNIX}:R>\` <t:${UNIX}:F>`)).toBe(false);
  });
});

describe("formatDiscordTime", () => {
  const at = (style: DiscordTimeStyle | undefined) => formatDiscordTime({ date, style });

  it("shows the date only for d and D, the time only for t and T, and both otherwise", () => {
    for (const style of ["d", "D", "f", "F", "s", "S", undefined] as const) expect(at(style)).toContain("2026");
    for (const style of ["t", "T"] as const) expect(at(style)).not.toContain("2026");
    expect(at("s").length).toBeGreaterThan(at("d").length);
  });

  it("shows seconds for T and S only", () => {
    for (const style of ["T", "S"] as const) expect(at(style)).toContain("25");
    for (const style of ["t", "s", "f", "F"] as const) expect(at(style)).not.toContain(":25");
  });

  it("reads no style as f", () => {
    expect(at(undefined)).toBe(at("f"));
  });

  it("spells a relative time out against now, either side of it", () => {
    const now = date.getTime();
    expect(formatDiscordTime({ date, style: "R" }, now)).toBe("0 seconds ago");
    expect(formatDiscordTime({ date, style: "R" }, now - 2 * 3_600_000)).toBe("in 2 hours");
    expect(formatDiscordTime({ date, style: "R" }, now + 60_000)).toBe("1 minute ago");
    expect(formatDiscordTime({ date, style: "R" }, now - 3 * 86_400_000)).toBe("in 3 days");
  });
});

describe("discordTimeIcs", () => {
  const now = Date.UTC(2026, 9, 24, 12, 0, 0);

  it("is a one-hour event at the instant, in UTC, with CRLF lines", () => {
    const ics = discordTimeIcs(date, "Release sync", now);
    expect(ics.split("\r\n")).toEqual([
      "BEGIN:VCALENDAR",
      "VERSION:2.0",
      "PRODID:-//Nexul//Timestamp//EN",
      "BEGIN:VEVENT",
      `UID:${UNIX * 1000}-${now}@nexul`,
      "DTSTAMP:20261024T120000Z",
      "DTSTART:20261024T183925Z",
      "DTEND:20261024T193925Z",
      "SUMMARY:Release sync",
      "END:VEVENT",
      "END:VCALENDAR",
      "",
    ]);
  });

  it("escapes the title's commas, semicolons, backslashes and line breaks", () => {
    expect(discordTimeIcs(date, "a, b; c\\d\ne", now)).toContain("SUMMARY:a\\, b\\; c\\\\d\\ne\r\n");
  });

  it("folds a long title onto continuation lines of at most 75 octets", () => {
    const lines = discordTimeIcs(date, "é".repeat(60), now).split("\r\n");
    const summary = lines.slice(lines.findIndex((l) => l.startsWith("SUMMARY:")), lines.indexOf("END:VEVENT"));
    expect(summary.length).toBeGreaterThan(1);
    for (const line of summary) expect(new TextEncoder().encode(line).length).toBeLessThanOrEqual(75);
    expect(summary.slice(1).every((l) => l.startsWith(" "))).toBe(true);
    expect(summary.map((l, i) => (i === 0 ? l : l.slice(1))).join("")).toBe(`SUMMARY:${"é".repeat(60)}`);
  });
});
