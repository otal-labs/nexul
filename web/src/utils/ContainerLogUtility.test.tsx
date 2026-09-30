import { describe, expect, it } from "vitest";

import { MAX_LOG_LINES, type ContainerLogLine, type ContainerLogWireLine } from "@/models/ContainerLog";
import { appendLogLines, chunkLogLines, formatContainerLogTime, looksLikeError } from "@/utils/ContainerLogUtility";

describe("looksLikeError", () => {
  it.each([
    ["ERROR database connection lost", true],
    ["request failed: error: timeout", true],
    ["FATAL: password authentication failed", true],
    ["Panic while handling request", true],
    ["CRITICAL disk almost full", true],
    ['time=12:00 level=error msg="boom"', true],
    ['time=12:00 level=fatal msg="boom"', true],
    ['{"level":"error","msg":"boom"}', true],
    ['{"level": "fatal","msg":"boom"}', true],
    ["panic: runtime error: index out of range", true],
    ["goroutine 187 [running]:", true],
    ["Traceback (most recent call last):", true],
    ["java.lang.IllegalStateException: closed", true],
    ["\tat com.example.Service.run(Service.java:42)", true],
    ["    at Object.<anonymous> (/app/index.js:3:9)", true],
    ["GET /healthz 200 3ms", false],
    ['{"level":"info","msg":"started"}', false],
    ["level=warn msg=slow", false],
    ["errors: 0", false],
    ["error_count=0", false],
    ["terror strikes", false],
    ["arrived at noon", false],
  ])("%s -> %s", (text, expected) => {
    expect(looksLikeError(text)).toBe(expected);
  });
});

const wire = (n: number): ContainerLogWireLine => ({ ts: `t${n}`, stream: "stdout", line: `l${n}` });
const line = (seq: number): ContainerLogLine => ({ seq, ts: `t${seq}`, stream: "stdout", text: `l${seq}` });

describe("appendLogLines", () => {
  it("keeps only the newest lines past the cap and keeps numbering upward", () => {
    const full = appendLogLines([], Array.from({ length: MAX_LOG_LINES }, (_, i) => wire(i)));
    const next = appendLogLines(full, [wire(MAX_LOG_LINES), wire(MAX_LOG_LINES + 1)]);
    expect(next).toHaveLength(MAX_LOG_LINES);
    expect(next[0]?.text).toBe("l2");
    expect(next.at(-1)).toMatchObject({ seq: MAX_LOG_LINES + 1, text: `l${MAX_LOG_LINES + 1}` });
  });
});

describe("formatContainerLogTime", () => {
  it("is blank for a line the daemon wrote without a timestamp", () => {
    expect(formatContainerLogTime({ ...line(1), ts: "" })).toBe("");
  });
});

describe("chunkLogLines", () => {
  it("keeps a chunk's lines when the oldest line is trimmed", () => {
    const lines = Array.from({ length: 120 }, (_, i) => line(i + 1));
    const before = chunkLogLines(lines);
    const after = chunkLogLines(lines.slice(1));
    expect(after[1]?.[1]).toEqual(before[1]?.[1]);
    expect(after[0]?.[1]).toHaveLength((before[0]?.[1].length ?? 0) - 1);
  });
});
