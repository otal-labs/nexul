import { spawnSync } from "node:child_process";
import type { AutomationsApi } from "./automations-api.ts";
import { log } from "./log.ts";
import type { Supervisor } from "./supervisor.ts";

export interface PollDeps {
  api: AutomationsApi;
  supervisor: Pick<Supervisor, "reconcile">;
  serverUrl: string;
  credential: string;
}

// pollOnce asks the instance what runs here and reconciles the workers; "removed" means this host was removed.
export async function pollOnce(deps: PollDeps): Promise<"running" | "removed"> {
  const result = await deps.api.fetchAssignments(deps.serverUrl, deps.credential);
  if (result.removed) return "removed";
  await deps.supervisor.reconcile(result.automations);
  return "running";
}

export type CommandRunner = (command: string, args: string[]) => number | null;

const runCommand: CommandRunner = (command, args) => spawnSync(command, args, { stdio: "inherit" }).status;

// uninstallSelf asks `nexul` to remove this host's service from outside it; the service stopping ends this process.
export function uninstallSelf(ctl: string, name: string, run: CommandRunner = runCommand): void {
  log("warn", "automations host was removed from the instance; uninstalling", { name });
  const status = run(ctl, ["uninstall", "automations", name, "--detach"]);
  if (status !== 0) log("error", "uninstall failed; remove the service by hand", { name, status });
}
