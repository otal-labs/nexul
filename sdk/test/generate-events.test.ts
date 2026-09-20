import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { CATALOG_GO_PATH, GENERATED_TS_PATH, extractCatalog, generateSource } from "../tools/generate-events.ts";

// The checked-in src/events.generated.ts must always match what the
// generator produces from the current catalog.go — this is the "regenerates
// and diffs so drift fails CI" gate the ticket calls for (AM11: schemas are
// the contract the SDK's types are generated from).
describe("generated events stay in sync with the catalog", () => {
  it("regenerating from catalog.go produces the checked-in file byte-for-byte", () => {
    const goSource = readFileSync(CATALOG_GO_PATH, "utf8");
    const catalog = extractCatalog(goSource);
    expect(catalog.size).toBeGreaterThan(0);
    const expected = generateSource(catalog);
    const actual = readFileSync(GENERATED_TS_PATH, "utf8");
    expect(actual).toBe(expected);
  });
});

describe("extractCatalog", () => {
  it("parses topic -> schema entries in source order", () => {
    const goSource = `
var catalogSchemas = map[string]string{
	"a.b": \`{"type":"object","properties":{"x":{"type":"string"}},"required":["x"]}\`,
	"c.d": \`{"type":"object","properties":{}}\`,
}`;
    const catalog = extractCatalog(goSource);
    expect([...catalog.keys()]).toEqual(["a.b", "c.d"]);
    expect(catalog.get("a.b")).toEqual({ type: "object", properties: { x: { type: "string" } }, required: ["x"] });
  });
});

describe("generateSource", () => {
  it("marks non-required fields optional and required fields as-is", () => {
    const catalog = extractCatalog(`"t.one": \`{"type":"object","required":["a"],"properties":{"a":{"type":"string"},"b":{"type":"integer"}}}\`,`);
    const source = generateSource(catalog);
    expect(source).toContain(`"a": string;`);
    expect(source).toContain(`"b"?: number;`);
  });

  it("renders enums as string literal unions", () => {
    const catalog = extractCatalog(`"t.two": \`{"type":"object","required":["s"],"properties":{"s":{"type":"string","enum":["open","closed"]}}}\`,`);
    const source = generateSource(catalog);
    expect(source).toContain(`"s": "open" | "closed";`);
  });

  it("produces a fixture value for every topic", () => {
    const catalog = extractCatalog(`"t.three": \`{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}\`,`);
    const source = generateSource(catalog);
    expect(source).toContain(`"t.three": {"id":"fixture-id"}`);
  });
});
