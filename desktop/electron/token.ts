// The client has no auth secret to verify the HMAC, so this is purely structural; probe.ts confirms reachability.

export interface TokenClaims {
  readonly instanceUrl: string;
  readonly expiresAtSec: number;
}

export type TokenParseResult =
  | { readonly ok: true; readonly claims: TokenClaims }
  | { readonly ok: false; readonly error: string };

export function parseConnectionToken(
  raw: string,
  nowSec = Math.floor(Date.now() / 1000),
): TokenParseResult {
  const parts = raw.split(".");
  if (parts.length !== 3) {
    return { ok: false, error: "not a connection token: expected three JWT parts" };
  }
  const [headerPart, payloadPart] = parts;
  if (headerPart === undefined || payloadPart === undefined) {
    return { ok: false, error: "not a connection token: expected three JWT parts" };
  }
  const headerJson = decodeBase64Url(headerPart);
  const payloadJson = decodeBase64Url(payloadPart);
  if (headerJson === null || payloadJson === null) {
    return { ok: false, error: "not a connection token: undecodable parts" };
  }
  const header = parseJson(headerJson);
  const claims = parseJson(payloadJson);
  if (header === null || claims === null) {
    return { ok: false, error: "not a connection token: parts are not JSON" };
  }
  if (header.alg !== "HS256" || header.typ !== "JWT") {
    return { ok: false, error: "not a connection token: unexpected header" };
  }
  const rawUrl = typeof claims.instance_url === "string" ? claims.instance_url.trim() : "";
  const instanceUrl = normalizeOrigin(rawUrl);
  if (instanceUrl === "") {
    return { ok: false, error: "connection token carries no usable instance URL" };
  }
  const expiresAtSec = typeof claims.exp === "number" ? claims.exp : NaN;
  if (!Number.isFinite(expiresAtSec) || nowSec >= expiresAtSec) {
    return { ok: false, error: "connection token has expired — generate a new one" };
  }
  return {
    ok: true,
    claims: { instanceUrl, expiresAtSec },
  };
}

// The shell only ever loads the origin, so this reduces to scheme://host[:port] ("" if not http(s)).
export function normalizeOrigin(url: string): string {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return "";
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return "";
  if (parsed.hostname === "") return "";
  return parsed.origin;
}

// FNV-1a: dependency-free and deterministic, so a re-import upserts the same instance, never a duplicate.
export function instanceIdFor(origin: string): string {
  let h = 0x811c9dc5;
  for (let i = 0; i < origin.length; i++) {
    h ^= origin.charCodeAt(i);
    h = (h * 0x01000193) >>> 0;
  }
  return h.toString(16).padStart(8, "0");
}

function decodeBase64Url(s: string): string | null {
  try {
    const b64 = s.replace(/-/g, "+").replace(/_/g, "/");
    const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
    return atob(padded);
  } catch {
    return null;
  }
}

function parseJson(s: string): Record<string, unknown> | null {
  try {
    const value = JSON.parse(s) as unknown;
    if (typeof value !== "object" || value === null || Array.isArray(value)) return null;
    return value as Record<string, unknown>;
  } catch {
    return null;
  }
}
