import { createHmac } from "node:crypto";

// Fixed identity for the seeded e2e user (see setup/global.ts); the token is forged with the same HMAC scheme the server uses (dev secret escape hatch, R1) so the browser session is a real, valid session.
export const E2E_UID = "e2e00000-0000-4000-8000-000000000001";
export const E2E_LOGIN = "e2e-user";
// second identity for real-time-collab presence (two distinct editors)
export const E2E_UID_2 = "e2e00000-0000-4000-8000-000000000002";
export const E2E_LOGIN_2 = "e2e-collab-user";
export const E2E_PROJECT_ID = "e2e00000-0000-4000-8000-000000000010";
export const E2E_PROJECT_NAME = "E2E Project";
// Pre-seeded default workspace (migration 0096) the e2e user is bound to as Owner; see setup/global.ts's workspace_members seed.
export const E2E_WORKSPACE_ID = "workspace-default";
// Id of that workspace's singleton Owner role (idx_roles_owner_singleton), shared between setup/global.ts's seed and setup/db.ts's onboarding-wizard reset/restore helpers so both sides use the same row.
export const E2E_OWNER_ROLE_ID = "role-workspace-default-owner";

export const AUTH_SECRET = process.env.NEXUL_AUTH_SECRET ?? "dev-secret-rotate-me";
export const BASE_URL = process.env.NEXUL_BASE_URL ?? "http://web";
export const MCP_URL = process.env.NEXUL_MCP_URL ?? `${BASE_URL}/mcp`;

// Mirrors the server's auth.Service.Sign: base64url({uid,exp}) + base64url(HMAC-SHA256(secret, payload)).
export function mintToken(uid: string = E2E_UID, ttlSec = 86400): string {
  const payload = Buffer.from(
    JSON.stringify({ uid, exp: Math.floor(Date.now() / 1000) + ttlSec }),
  ).toString("base64url");
  const mac = createHmac("sha256", AUTH_SECRET).update(payload).digest("base64url");
  return `${payload}.${mac}`;
}

// localStorage shape zustand persist expects for the session store (key "session", partialize {token, isLoggedIn}).
export function sessionLocalStorageValue(token: string) {
  return JSON.stringify({ state: { token, isLoggedIn: true }, version: 0 });
}

export const authedHeaders = (token: string) => ({
  Authorization: `Bearer ${token}`,
  "Content-Type": "application/json",
});

// An MCP client must accept both a JSON body and an event stream, or the server answers 400.
export const mcpHeaders = (token: string) => ({
  ...authedHeaders(token),
  Accept: "application/json, text/event-stream",
});

// Shared HTTP-gateway fetch client for specs hitting `/api/...` directly instead of driving the browser; each spec mints its own token and builds its own client so a 401 test can still hit the gateway unauthed.
export function makeApiClient(token: string) {
  return async function api<T = unknown>(
    path: string,
    init: RequestInit = {},
  ): Promise<{ status: number; body: T | null }> {
    const res = await fetch(`${BASE_URL}${path}`, {
      ...init,
      headers: { ...authedHeaders(token), ...(init.headers ?? {}) },
    });
    const text = await res.text();
    return { status: res.status, body: text ? (JSON.parse(text) as T) : null };
  };
}
