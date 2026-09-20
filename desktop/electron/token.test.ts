import { describe, expect, it } from "vitest";

import { instanceIdFor, normalizeOrigin, parseConnectionToken } from "./token";

const b64url = (obj: unknown): string =>
  btoa(JSON.stringify(obj)).replace(/=+$/, "").replace(/\+/g, "-").replace(/\//g, "_");

const makeToken = (
  claims: Record<string, unknown>,
  header: Record<string, unknown> = { alg: "HS256", typ: "JWT" },
): string => `${b64url(header)}.${b64url(claims)}.c2ln`;

const NOW = 1_750_000_000;

describe("parseConnectionToken", () => {
  it.each([
    ["accepts a well-formed token", makeToken({ instance_url: "https://deploy.example.com", exp: NOW + 3600, iat: NOW, version: 1 }), true],
    ["rejects a token split into two parts", "a.b", false],
    ["rejects a token with four parts", "a.b.c.d", false],
    ["rejects a token with one part", "a", false],
    ["rejects an empty string", "", false],
    ["rejects an undecodable payload", `${b64url({ alg: "HS256", typ: "JWT" })}.not-base64@@@.sig`, false],
    ["rejects a non-JSON payload", `${b64url({ alg: "HS256", typ: "JWT" })}.${btoa("hello").replace(/=+$/, "")}.sig`, false],
    ["rejects a non-HS256 header", makeToken({ instance_url: "https://x.example", exp: NOW + 3600 }, { alg: "none", typ: "JWT" }), false],
    ["rejects a payload with a non-JWT type", makeToken({ instance_url: "https://x.example", exp: NOW + 3600 }, { alg: "HS256", typ: "JWE" }), false],
    ["rejects a missing instance_url", makeToken({ exp: NOW + 3600, iat: NOW, version: 1 }), false],
    ["rejects a non-http(s) instance_url", makeToken({ instance_url: "ftp://files.example", exp: NOW + 3600 }), false],
    ["rejects an unparseable instance_url", makeToken({ instance_url: "not a url", exp: NOW + 3600 }), false],
    ["rejects a non-numeric exp", makeToken({ instance_url: "https://x.example", exp: "soon" }), false],
    ["rejects an expired token", makeToken({ instance_url: "https://x.example", exp: NOW - 1 }), false],
    ["rejects a token expiring right now", makeToken({ instance_url: "https://x.example", exp: NOW }), false],
  ] as const)("%s", (_name, token, expected) => {
    expect(parseConnectionToken(token, NOW).ok).toBe(expected);
  });

  it("normalizes the instance URL to its origin", () => {
    const result = parseConnectionToken(
      makeToken({ instance_url: "  https://Deploy.Example.com:8443/some/path?q=1  ", exp: NOW + 3600 }),
      NOW,
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.claims.instanceUrl).toBe("https://deploy.example.com:8443");
    }
  });

  it("reports the reason for rejection", () => {
    const result = parseConnectionToken(makeToken({ instance_url: "https://x.example", exp: NOW - 1 }), NOW);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("expired");
  });
});

describe("normalizeOrigin", () => {
  it.each([
    ["https://deploy.example.com", "https://deploy.example.com"],
    ["http://localhost:8080", "http://localhost:8080"],
    ["https://x.example:8443/", "https://x.example:8443"],
    ["https://user:pass@x.example/path", "https://x.example"],
  ] as const)("%s → %s", (input, expected) => {
    expect(normalizeOrigin(input)).toBe(expected);
  });

  it.each([
    "ftp://files.example",
    "not a url",
    "",
    "//host-only",
    "javascript:alert(1)",
  ])("rejects %s", (input) => {
    expect(normalizeOrigin(input)).toBe("");
  });
});

describe("instanceIdFor", () => {
  it("is stable across calls for the same origin", () => {
    expect(instanceIdFor("https://a.example")).toBe(instanceIdFor("https://a.example"));
  });

  it("differs between origins", () => {
    expect(instanceIdFor("https://a.example")).not.toBe(instanceIdFor("https://b.example"));
  });
});
