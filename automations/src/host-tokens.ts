import { existsSync, readFileSync } from "node:fs";

export interface AutomationTarget {
  id: string;
  name: string;
  token: string;
}

function isTarget(v: unknown): v is AutomationTarget {
  const t = v as Partial<AutomationTarget> | null;
  return typeof t?.id === "string" && typeof t?.name === "string" && typeof t?.token === "string";
}

// readHostTokens loads the file the server's seeder writes on-disk for
// default automations (internal/automations/host_tokens.go) — the pragmatic
// v1 bootstrap: a fresh install lands with the board pair runnable with no
// manual token handoff, at the cost of the host only knowing about
// automations someone put in this file.
export function readHostTokens(path: string): AutomationTarget[] {
  if (!existsSync(path)) return [];
  const raw: unknown = JSON.parse(readFileSync(path, "utf8"));
  if (!Array.isArray(raw)) throw new Error(`${path} must contain a JSON array`);
  return raw.filter(isTarget);
}
