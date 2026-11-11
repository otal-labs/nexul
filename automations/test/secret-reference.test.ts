import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { parseSecretReference, type ReferenceKind } from "../src/secret-reference.ts";

// The Go parser's own table is the contract; reading it here means the two cannot drift apart unnoticed.
const goTable = readFileSync(`${import.meta.dir}/../../internal/platform/secretref/secretref_test.go`, "utf8");
const row = /^\s*\{"([^"]*)", ("(?:[^"\\]|\\.)*"), (Literal|Reference|Malformed), Ref\{(.*)\}\},$/gm;

const cases = [...goTable.matchAll(row)].map(([, name = "", value = "", kind = "", fields = ""]) => ({
  name,
  value: JSON.parse(value) as string,
  kind: kind.toLowerCase() as ReferenceKind,
  ref: Object.fromEntries([...fields.matchAll(/(\w+): "([^"]*)"/g)].map(([, k = "", v = ""]) => [k.toLowerCase(), v])),
}));

describe("parseSecretReference", () => {
  test("the Go table was read in full", () => {
    expect(cases.length).toBe(14);
  });

  for (const c of cases) {
    test(`agrees with the Go parser: ${c.name}`, () => {
      const got = parseSecretReference(c.value);
      expect(got.kind).toBe(c.kind);
      expect(got.ref ?? {}).toEqual(c.ref);
    });
  }
});
