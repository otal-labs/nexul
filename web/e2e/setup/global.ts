import { Database } from "bun:sqlite";

import {
  BASE_URL,
  E2E_LOGIN,
  E2E_LOGIN_2,
  E2E_OWNER_ROLE_ID,
  E2E_PROJECT_ID,
  E2E_PROJECT_NAME,
  E2E_UID,
  E2E_UID_2,
  E2E_WORKSPACE_ID,
} from "../helpers";

// Mirrors what auth.Service.CompleteOwnerWizard / tenancy.BindDefaultWorkspaceOwner set up for a real owner, since this seed bypasses that flow and writes the DB directly; workspace membership (not can_create_workspace) is what grants doc/board access via access.Service.HasPermission's Owner-role bypass (ticket 11).
const DEFAULT_WORKSPACE_ID = E2E_WORKSPACE_ID;
const DEFAULT_OWNER_ROLE_ID = E2E_OWNER_ROLE_ID;

// Seeds a deterministic e2e owner (first_login_done so OnboardingGate lets the app load), bound to the default workspace since doc/board access now flows through workspace membership rather than can_create_workspace (ticket 11); default statuses ship in migration 0022 so the board works out of the box.
export async function setup(): Promise<void> {
  const dbPath = process.env.NEXUL_E2E_DB ?? "/data/nexul.db";

  let db: Database | undefined;
  for (let i = 0; i < 120; i++) {
    try {
      db = new Database(dbPath, { readwrite: true });
      const row = db
        .query("SELECT version FROM schema_migrations WHERE version = ?")
        .get("0105_workspace_invites");
      if (row) break;
    } catch {
      // DB file doesn't exist yet or is mid-migration — retry.
    }
    db?.close();
    db = undefined;
    Bun.sleepSync(500);
  }
  if (!db) throw new Error("timed out waiting for the e2e DB to migrate");

  const now = Math.floor(Date.now() / 1000);
  db.run(
    `INSERT INTO users (id, provider, provider_user_id, login, name, can_create_workspace, first_login_done, created_at, updated_at)
     VALUES (?, 'github', '0', ?, 'E2E User', 1, 1, ?, ?)
     ON CONFLICT(provider, provider_user_id)
     DO UPDATE SET can_create_workspace = 1, first_login_done = 1, login = excluded.login`,
    [E2E_UID, E2E_LOGIN, now, now],
  );
  // second user for real-time-collab presence (distinct editor identity)
  db.run(
    `INSERT INTO users (id, provider, provider_user_id, login, name, can_create_workspace, first_login_done, created_at, updated_at)
     VALUES (?, 'github', '1', ?, 'Collab Partner', 0, 1, ?, ?)
     ON CONFLICT(provider, provider_user_id)
     DO UPDATE SET can_create_workspace = 0, first_login_done = 1, login = excluded.login`,
    [E2E_UID_2, E2E_LOGIN_2, now, now],
  );
  db.run(
    `INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at)
     VALUES (?, ?, 0, ?, ?, ?)
     ON CONFLICT(id) DO UPDATE SET name = excluded.name, workspace_id = excluded.workspace_id`,
    [E2E_PROJECT_ID, E2E_PROJECT_NAME, DEFAULT_WORKSPACE_ID, now, now],
  );
  // Bind E2E_UID as the default workspace's Owner; idempotent because the singleton Owner-role unique index means only insert if this workspace has none yet.
  db.run(
    `INSERT INTO roles (id, workspace_id, name, permission_mask, is_owner_role, created_at, updated_at)
     SELECT ?, ?, 'Owner', 0, 1, ?, ?
     WHERE NOT EXISTS (SELECT 1 FROM roles WHERE workspace_id = ? AND is_owner_role = 1)`,
    [DEFAULT_OWNER_ROLE_ID, DEFAULT_WORKSPACE_ID, now, now, DEFAULT_WORKSPACE_ID],
  );
  db.run(
    `INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at)
     VALUES (?, ?, (SELECT id FROM roles WHERE workspace_id = ? AND is_owner_role = 1), ?)
     ON CONFLICT(user_id, workspace_id)
     DO UPDATE SET role_id = excluded.role_id`,
    [E2E_UID, DEFAULT_WORKSPACE_ID, DEFAULT_WORKSPACE_ID, now],
  );
  db.close();
  console.log("e2e seed: user + project + workspace membership ready");

  await bootstrapInstance();
}

// Drives the public bootstrap endpoint so the AU7 gate in AppRouter never blocks the specs; checks bootstrap-status first because run-e2e.sh's two vitest phases share one DB volume and Service.Bootstrap rejects a second call once configured, so this must be a no-op on the second phase.
async function bootstrapInstance(): Promise<void> {
  // web (nginx) has no depends_on-driven readiness gate in docker-compose.e2e.yml, so retry past a cold-start connection refusal like the DB-migration wait above.
  let statusRes: Response | undefined;
  for (let i = 0; i < 20; i++) {
    try {
      statusRes = await fetch(`${BASE_URL}/api/auth/bootstrap-status`);
      break;
    } catch {
      await new Promise((r) => setTimeout(r, 500));
    }
  }
  if (!statusRes) throw new Error("timed out waiting for the web container to accept connections");
  const status = (await statusRes.json()) as { configured: boolean };
  if (status.configured) return;

  const res = await fetch(`${BASE_URL}/api/auth/bootstrap`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      instance_url: BASE_URL,
      client_id: "e2e-github-client-id",
      client_secret: "e2e-github-client-secret",
      app_slug: "e2e-github-app",
    }),
  });
  if (!res.ok) {
    throw new Error(`bootstrap failed: ${res.status} ${await res.text()}`);
  }
  console.log("e2e seed: instance bootstrapped");
}

export function teardown(): void {
  // nothing to do; the compose volume is discarded with `down -v`
}
