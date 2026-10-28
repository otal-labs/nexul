import { createHash, randomBytes, randomUUID } from "node:crypto";
import { DatabaseSync } from "node:sqlite";

// Fixed identity for the seeded e2e user (see setup/global.ts); sessions are rows the server looks up by token hash, so mintToken writes a real one.
export const E2E_UID = "e2e00000-0000-4000-8000-000000000001";
export const E2E_LOGIN = "e2e-user";
// second identity for real-time-collab presence (two distinct editors)
export const E2E_UID_2 = "e2e00000-0000-4000-8000-000000000002";
export const E2E_LOGIN_2 = "e2e-collab-user";
export const E2E_PROJECT_ID = "e2e00000-0000-4000-8000-000000000010";
export const E2E_PROJECT_NAME = "E2E Project";
// Pre-seeded default workspace (migration 0096) the e2e user is bound to as Owner; see setup/global.ts's workspace_members seed.
export const E2E_WORKSPACE_ID = "workspace-default";
// The machine the compose stack's runner enrolls as, which every stack the specs create names.
export const E2E_MACHINE = "instance";
// Every workspace page lives under its slug, e.g. `${E2E_WORKSPACE_PATH}/board`.
export const E2E_WORKSPACE_PATH = "/default";
// Id of that workspace's singleton Owner role (idx_roles_owner_singleton), shared between setup/global.ts's seed and setup/db.ts's onboarding-wizard reset/restore helpers so both sides use the same row.
export const E2E_OWNER_ROLE_ID = "role-workspace-default-owner";

const E2E_DB_PATH = process.env.NEXUL_E2E_DB ?? "/data/nexul.db";
export const AUTH_SECRET = process.env.NEXUL_AUTH_SECRET ?? "dev-secret-rotate-me";
export const BASE_URL = process.env.NEXUL_BASE_URL ?? "http://web";
export const MCP_URL = process.env.NEXUL_MCP_URL ?? `${BASE_URL}/mcp`;

// The server writes the same file, so wait out its locks instead of failing with "database is locked".
export const openDb = () => new DatabaseSync(E2E_DB_PATH, { timeout: 10_000 });

// Mirrors auth.Service.CreateSession: a ses_ token whose sha256 is stored in the sessions table.
export function mintToken(uid: string = E2E_UID, ttlSec = 86400): string {
  const raw = `ses_${randomBytes(32).toString("base64url")}`;
  const now = Math.floor(Date.now() / 1000);
  const db = openDb();
  db.prepare(
    `INSERT INTO sessions (id, user_id, token_hash, client, created_at, last_active_at, expires_at)
     VALUES (?, ?, ?, 'browser', ?, ?, ?)`,
  ).run(randomUUID(), uid, createHash("sha256").update(raw).digest("hex"), now, now, now + ttlSec);
  db.close();
  return raw;
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

// A ticket's status is the id of one of its project's board columns, not a fixed word.
export async function progressStatusId(api: ReturnType<typeof makeApiClient>): Promise<string> {
  const { body } = await api<{ id: string; kind: string }[]>(`/api/statuses?project_id=${E2E_PROJECT_ID}`);
  return body!.find((s) => s.kind === "progress")!.id;
}
