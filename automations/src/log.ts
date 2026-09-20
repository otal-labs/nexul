// Structured logs only (AGENTS.md hard rule 8): one JSON line per event on
// stdout, no console.log in the rest of the codebase.
export type Level = "info" | "warn" | "error";

export function log(level: Level, message: string, meta: Record<string, unknown> = {}): void {
  const line = JSON.stringify({ ts: new Date().toISOString(), level, message, ...meta });
  if (level === "error") {
    console.error(line);
    return;
  }
  console.log(line);
}
