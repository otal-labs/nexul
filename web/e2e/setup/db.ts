import { Database } from "bun:sqlite";

import { E2E_OWNER_ROLE_ID, E2E_PROJECT_ID, E2E_UID, E2E_WORKSPACE_ID } from "../helpers";

const DB_PATH = process.env.NEXUL_E2E_DB ?? "/data/nexul.db";

// Flips onboarding flags on the seeded user so specs can exercise the wizards through the real UI; the wizards restore the flags on completion, this exists only as a safety net for cleanup when a test fails midway.
export function updateUser(fields: { can_create_workspace?: number; first_login_done?: number }): void {
  const db = new Database(DB_PATH, { readwrite: true });
  const sets: string[] = [];
  const values: number[] = [];
  if (fields.can_create_workspace !== undefined) {
    sets.push("can_create_workspace = ?");
    values.push(fields.can_create_workspace);
  }
  if (fields.first_login_done !== undefined) {
    sets.push("first_login_done = ?");
    values.push(fields.first_login_done);
  }
  if (sets.length === 0) {
    db.close();
    return;
  }
  db.run(`UPDATE users SET ${sets.join(", ")} WHERE id = ?`, ...values, E2E_UID);
  db.close();
}

// Removes the seeded Owner role/membership so the owner-wizard spec can exercise the real tenancy.Service.BindDefaultWorkspaceOwner flow, which always INSERTs a fresh singleton Owner role rather than reusing the one global.ts seeds for every other spec's benefit; leaving that row in place makes CompleteOwnerWizard 409 with a role conflict on a genuinely fresh onboarding run.
export function clearDefaultWorkspaceOwnership(): void {
  const db = new Database(DB_PATH, { readwrite: true });
  db.run(`DELETE FROM workspace_members WHERE user_id = ? AND workspace_id = ?`, [E2E_UID, E2E_WORKSPACE_ID]);
  db.run(`DELETE FROM roles WHERE workspace_id = ? AND is_owner_role = 1`, [E2E_WORKSPACE_ID]);
  db.close();
}

// Re-seeds what clearDefaultWorkspaceOwnership removed, as a safety net for afterAll so a mid-test failure can't leave every other spec file without a workspace owner; idempotent like global.ts's seed, a no-op once the wizard has already recreated them.
export function restoreDefaultWorkspaceOwnership(): void {
  const db = new Database(DB_PATH, { readwrite: true });
  const now = Math.floor(Date.now() / 1000);
  db.run(
    `INSERT INTO roles (id, workspace_id, name, permission_mask, is_owner_role, created_at, updated_at)
     SELECT ?, ?, 'Owner', 0, 1, ?, ?
     WHERE NOT EXISTS (SELECT 1 FROM roles WHERE workspace_id = ? AND is_owner_role = 1)`,
    [E2E_OWNER_ROLE_ID, E2E_WORKSPACE_ID, now, now, E2E_WORKSPACE_ID],
  );
  db.run(
    `INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at)
     VALUES (?, ?, (SELECT id FROM roles WHERE workspace_id = ? AND is_owner_role = 1), ?)
     ON CONFLICT(user_id, workspace_id)
     DO UPDATE SET role_id = excluded.role_id`,
    [E2E_UID, E2E_WORKSPACE_ID, E2E_WORKSPACE_ID, now],
  );
  db.close();
}

// Docs are project-scoped (internal/docs/usecase.go rejects a missing project_id), so this always creates the doc under the seeded e2e project.
export async function createDocWith(token: string, title: string): Promise<string> {
  const res = await fetch(`${process.env.NEXUL_BASE_URL ?? "http://web"}/api/docs`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ title, project_id: E2E_PROJECT_ID }),
  });
  const body = (await res.json()) as { id: string };
  return body.id;
}
