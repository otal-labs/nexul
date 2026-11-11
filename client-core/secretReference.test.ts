import { describe, expect, it } from "vitest";

import { holderNamedBy, parseSecretReference, type ParsedSecret } from "./secretReference";

const reference = (vault: string, item: string, field: string, section?: string): ParsedSecret => ({
  kind: "reference",
  ref: { scheme: "op", vault, item, field, ...(section && { section }) },
});

// The same table as internal/platform/secretref/secretref_test.go, so the two parsers cannot drift apart.
describe("parseSecretReference", () => {
  it.each<[string, string, ParsedSecret]>([
    ["three parts", "op://acme/db/password", reference("acme", "db", "password")],
    ["with section", "op://acme/db/prod/password", reference("acme", "db", "password", "prod")],
    ["spaces in names", "op://Acme Ops/Prod DB/password", reference("Acme Ops", "Prod DB", "password")],
    ["postgres url", "postgres://u:p@example.com/db", { kind: "literal" }],
    ["https url", "https://example.com/a/b/c", { kind: "literal" }],
    ["embedded", "x op://acme/db/password", { kind: "literal" }],
    ["plain value", "hunter2", { kind: "literal" }],
    ["empty", "", { kind: "literal" }],
    ["vault only", "op://acme", { kind: "malformed" }],
    ["two parts", "op://acme/db", { kind: "malformed" }],
    ["empty vault", "op:///db/password", { kind: "malformed" }],
    ["empty field", "op://acme/db/", { kind: "malformed" }],
    ["too many parts", "op://a/b/c/d/e", { kind: "malformed" }],
    ["part padded with a zero-width no-break space", "op://acme/db/﻿password", reference("acme", "db", "﻿password")],
    ["trailing newline","op://acme/db/password\n", { kind: "malformed" }],
  ])("%s", (_name, value, want) => {
    expect(parseSecretReference(value)).toEqual(want);
  });
});

describe("holderNamedBy", () => {
  const refusal = "DB_URL is not a valid secret reference; it must read op://vault/item/field or op://vault/item/section/field";

  it("finds the field the refusal names, and no other with the same start", () => {
    expect(holderNamedBy(refusal, ["DB", "DB_URL"])).toBe("DB_URL");
  });

  it("leaves any other refusal unmatched", () => {
    expect(holderNamedBy("DB_URL is required", ["DB_URL"])).toBeUndefined();
  });
});
