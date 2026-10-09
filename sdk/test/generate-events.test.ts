import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { CATALOG_PATH, GENERATED_TS_PATH, generateSource, readCatalog } from "../tools/generate-events.ts";

// The checked-in src/events.generated.ts must always match what the generator produces from the published contract,
// so a contract change without a regenerate fails CI.
describe("generated events stay in sync with the catalog", () => {
  it("regenerating from the contract file produces the checked-in file byte-for-byte", () => {
    const catalog = readCatalog(readFileSync(CATALOG_PATH, "utf8"));
    expect(catalog.size).toBeGreaterThan(0);
    const expected = generateSource(catalog);
    const actual = readFileSync(GENERATED_TS_PATH, "utf8");
    expect(actual).toBe(expected);
  });
});

describe("generateSource", () => {
  it("marks non-required fields optional and required fields as-is", () => {
    const catalog = readCatalog(`{"t.one":{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"integer"}}}}`);
    const source = generateSource(catalog);
    expect(source).toContain(`"a": string;`);
    expect(source).toContain(`"b"?: number;`);
  });

  it("renders enums as string literal unions", () => {
    const catalog = readCatalog(`{"t.two":{"type":"object","required":["s"],"properties":{"s":{"type":"string","enum":["open","closed"]}}}}`);
    const source = generateSource(catalog);
    expect(source).toContain(`"s": "open" | "closed";`);
  });

  it("wraps a list of enum values so the list is the union's", () => {
    const catalog = readCatalog(`{"t.five":{"type":"object","properties":{"c":{"type":"array","items":{"type":"string","enum":["a","b"]}}}}}`);
    expect(generateSource(catalog)).toContain(`"c"?: ("a" | "b")[];`);
  });

  it("produces a fixture value for every topic", () => {
    const catalog = readCatalog(`{"t.three":{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}}`);
    const source = generateSource(catalog);
    expect(source).toContain(`"t.three": {"id":"fixture-id"}`);
  });

  it("types a nullable list as a list or null, and fills its fixture with an element", () => {
    const catalog = readCatalog(`{"t.four":{"type":"object","required":["ids"],"properties":{"ids":{"type":["null","array"],"items":{"type":"string"}}}}}`);
    const source = generateSource(catalog);
    expect(source).toContain(`"ids": null | string[];`);
    expect(source).toContain(`"t.four": {"ids":["fixture-ids"]}`);
  });
});
