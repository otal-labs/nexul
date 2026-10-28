import { createCipheriv, createHash, randomBytes, randomUUID } from "node:crypto";
import { setTimeout as sleep } from "node:timers/promises";

import {
  AUTH_SECRET,
  BASE_URL,
  E2E_LOGIN,
  E2E_LOGIN_2,
  E2E_MACHINE,
  E2E_OWNER_ROLE_ID,
  E2E_PROJECT_ID,
  E2E_PROJECT_NAME,
  E2E_UID,
  E2E_UID_2,
  E2E_WORKSPACE_ID,
  mintToken,
  openDb,
} from "../helpers";

// Mirrors what auth.Service.CompleteOwnerWizard / tenancy.BindDefaultWorkspaceOwner set up for a real owner, since this seed bypasses that flow and writes the DB directly; workspace membership is what grants every permission via access.Service.HasPermission's Owner-role bypass.
const DEFAULT_WORKSPACE_ID = E2E_WORKSPACE_ID;
const DEFAULT_OWNER_ROLE_ID = E2E_OWNER_ROLE_ID;

// Seeds a deterministic e2e owner (first_login_done so OnboardingGate lets the app load), bound to the default workspace since every permission flows through workspace membership; default statuses ship in migration 0022 so the board works out of the box.
export async function setup(): Promise<void> {
  // The server migrates the DB before it listens, so a live gateway means the schema is complete.
  await waitForServer();
  const db = openDb();

  const now = Math.floor(Date.now() / 1000);
  db.prepare(
    `INSERT INTO users (id, login, name, first_login_done, created_at, updated_at)
     VALUES (?, ?, 'E2E User', 1, ?, ?)
     ON CONFLICT(id)
     DO UPDATE SET first_login_done = 1, login = excluded.login`,
  ).run(E2E_UID, E2E_LOGIN, now, now);
  db.prepare(
    `INSERT INTO user_identities (user_id, provider, provider_user_id, login, name, created_at)
     VALUES (?, 'github', '0', ?, 'E2E User', ?)
     ON CONFLICT(provider, provider_user_id)
     DO UPDATE SET user_id = excluded.user_id, login = excluded.login`,
  ).run(E2E_UID, E2E_LOGIN, now);
  // second user for real-time-collab presence (distinct editor identity)
  db.prepare(
    `INSERT INTO users (id, login, name, first_login_done, created_at, updated_at)
     VALUES (?, ?, 'Collab Partner', 1, ?, ?)
     ON CONFLICT(id)
     DO UPDATE SET first_login_done = 1, login = excluded.login`,
  ).run(E2E_UID_2, E2E_LOGIN_2, now, now);
  db.prepare(
    `INSERT INTO user_identities (user_id, provider, provider_user_id, login, name, created_at)
     VALUES (?, 'github', '1', ?, 'Collab Partner', ?)
     ON CONFLICT(provider, provider_user_id)
     DO UPDATE SET user_id = excluded.user_id, login = excluded.login`,
  ).run(E2E_UID_2, E2E_LOGIN_2, now);
  db.prepare(
    `INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at)
     VALUES (?, ?, 0, ?, ?, ?)
     ON CONFLICT(id) DO UPDATE SET name = excluded.name, workspace_id = excluded.workspace_id`,
  ).run(E2E_PROJECT_ID, E2E_PROJECT_NAME, DEFAULT_WORKSPACE_ID, now, now);
  // What creating a project through the product seeds, which this raw insert bypasses: the board's statuses and ticket types, and the default folder a new doc lands in.
  if (!db.prepare(`SELECT 1 FROM statuses WHERE project_id = ?`).get(E2E_PROJECT_ID)) {
    const columns: [string, string][] = [["Backlog", "backlog"], ["In progress", "progress"], ["In review", "review"], ["Testing", "testing"], ["Done", "done"]];
    for (const [i, [name, kind]] of columns.entries()) {
      db.prepare(
        `INSERT INTO statuses (id, project_id, name, position, kind, icon, created_at, updated_at) VALUES (?, ?, ?, ?, ?, '', ?, ?)`,
      ).run(randomUUID(), E2E_PROJECT_ID, name, i, kind, now, now);
    }
  }
  if (!db.prepare(`SELECT 1 FROM ticket_types WHERE project_id = ?`).get(E2E_PROJECT_ID)) {
    for (const [i, name] of ["task", "bug", "feature"].entries()) {
      db.prepare(
        `INSERT INTO ticket_types (id, project_id, name, position, color, body_template, created_at, updated_at) VALUES (?, ?, ?, ?, '', '', ?, ?)`,
      ).run(randomUUID(), E2E_PROJECT_ID, name, i, now, now);
    }
  }
  db.prepare(
    `INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at) VALUES (?, ?, 'Main', 1, ?, ?)
     ON CONFLICT DO NOTHING`,
  ).run(randomUUID(), E2E_PROJECT_ID, now, now);
  // Bind E2E_UID as the default workspace's Owner; idempotent because the singleton Owner-role unique index means only insert if this workspace has none yet.
  db.prepare(
    `INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at)
     SELECT ?, ?, 'Owner', 1, ?, ?
     WHERE NOT EXISTS (SELECT 1 FROM roles WHERE workspace_id = ? AND is_owner_role = 1)`,
  ).run(DEFAULT_OWNER_ROLE_ID, DEFAULT_WORKSPACE_ID, now, now, DEFAULT_WORKSPACE_ID);
  db.prepare(
    `INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES (?, ?, ?)
     ON CONFLICT(user_id, workspace_id) DO NOTHING`,
  ).run(E2E_UID, DEFAULT_WORKSPACE_ID, now);
  // The collab partner is a plain member with no role, so a doc is theirs only once shared.
  db.prepare(
    `INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES (?, ?, ?)
     ON CONFLICT(user_id, workspace_id) DO NOTHING`,
  ).run(E2E_UID_2, DEFAULT_WORKSPACE_ID, now);
  db.prepare(`DELETE FROM member_roles WHERE user_id = ? AND workspace_id = ?`).run(E2E_UID, DEFAULT_WORKSPACE_ID);
  db.prepare(
    `INSERT INTO member_roles (user_id, workspace_id, role_id, created_at)
     SELECT ?, ?, id, ? FROM roles WHERE workspace_id = ? AND is_owner_role = 1`,
  ).run(E2E_UID, DEFAULT_WORKSPACE_ID, now, DEFAULT_WORKSPACE_ID);
  // POST /api/auth/bootstrap needs a setup pass, which exists only while no user does, and verifies the GitHub App against github.com, so the instance is configured here instead so the first-run gate never blocks the specs.
  db.prepare(
    `UPDATE instance_settings SET instance_url = ?, github_oauth_client_id = ?, github_oauth_client_secret = ? WHERE id = 1`,
  ).run(BASE_URL, "e2e-github-client-id", sealSecret("e2e-github-client-secret"));
  db.close();
  console.log("e2e seed: user + project + workspace membership + instance settings ready");
  await waitForMachine();
}

// The server has no readiness gate in docker-compose.e2e.yml, so retry past a cold-start connection refusal.
async function waitForServer(): Promise<void> {
  for (let i = 0; i < 120; i++) {
    try {
      await fetch(`${BASE_URL}/api/auth/bootstrap-status`);
      return;
    } catch {
      await sleep(500);
    }
  }
  throw new Error("timed out waiting for the server to accept connections");
}

// The runner enrolls after the server boots, and a stack names the machine it deploys to, so specs wait for it to be listed.
async function waitForMachine(): Promise<void> {
  const headers = { Authorization: `Bearer ${mintToken()}` };
  for (let i = 0; i < 120; i++) {
    const res = await fetch(`${BASE_URL}/api/machines/deployable`, { headers });
    const machines = (await res.json()) as { name: string }[];
    if (machines.some((m) => m.name === E2E_MACHINE)) return;
    await sleep(500);
  }
  throw new Error(`timed out waiting for machine ${E2E_MACHINE}`);
}

// Mirrors crypto.Encrypt: base64(nonce || AES-256-GCM(sha256(secret)) ciphertext || tag), unpadded.
function sealSecret(plain: string): string {
  const nonce = randomBytes(12);
  const cipher = createCipheriv("aes-256-gcm", createHash("sha256").update(AUTH_SECRET).digest(), nonce);
  const sealed = Buffer.concat([nonce, cipher.update(plain, "utf8"), cipher.final(), cipher.getAuthTag()]);
  return sealed.toString("base64").replace(/=+$/, "");
}

export function teardown(): void {
  // nothing to do; the compose volume is discarded with `down -v`
}
