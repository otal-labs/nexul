import { describe, expect, test } from "bun:test";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { readHostTokens } from "../src/host-tokens.ts";

function tempFile(content: string): string {
  const dir = mkdtempSync(join(tmpdir(), "host-tokens-test-"));
  const file = join(dir, "tokens.json");
  writeFileSync(file, content);
  return file;
}

describe("readHostTokens", () => {
  test("returns an empty list when the file doesn't exist", () => {
    expect(readHostTokens(join(tmpdir(), "does-not-exist.json"))).toEqual([]);
  });

  test("parses well-formed entries", () => {
    const file = tempFile(JSON.stringify([{ id: "a1", name: "Ticket finished", token: "dat_1" }]));
    expect(readHostTokens(file)).toEqual([{ id: "a1", name: "Ticket finished", token: "dat_1" }]);
  });

  test("drops malformed entries instead of crashing the host", () => {
    const file = tempFile(JSON.stringify([{ id: "a1", name: "ok", token: "dat_1" }, { id: "a2" }, "garbage"]));
    expect(readHostTokens(file)).toEqual([{ id: "a1", name: "ok", token: "dat_1" }]);
  });

  test("rejects a file that isn't a JSON array", () => {
    const file = tempFile(JSON.stringify({ id: "a1" }));
    expect(() => readHostTokens(file)).toThrow();
  });
});
