import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";
import type { ContainerLogSource } from "@/models/ContainerLog";

export type CommandStatus = "queued" | "running" | "succeeded" | "failed" | "timed_out" | "cancelled" | "refused";

// Mirrors shelljob.Job: one command run through a runner, on a computer (computer_id, owner_id) or on a machine
// (runner_id, machine_id), kept with who ran it.
export interface ComputerCommand {
  id: string;
  computer_id?: string;
  owner_id?: string;
  runner_id?: string;
  machine_id?: string;
  // The runner a machine's command ran on, on a list.
  runner_name?: string;
  requester_id: string;
  // Who ran it, on a list; the owner's list names each person a share let run one.
  requester_name?: string;
  adapter: "web" | "mcp";
  shell: string;
  command: string;
  cwd: string;
  timeout_seconds: number;
  status: CommandStatus;
  exit_code: number | null;
  error: string;
  output_truncated: boolean;
  output_purged: boolean;
  created_at: string;
  started_at: string | null;
  ended_at: string | null;
}

// One page of a computer's commands, newest first, without their output.
export interface ComputerCommandPage {
  items: ComputerCommand[];
  total: number;
}

const tones: Record<CommandStatus, SettingsStatusTone> = {
  queued: "muted",
  running: "warning",
  succeeded: "success",
  failed: "destructive",
  timed_out: "destructive",
  cancelled: "muted",
  refused: "destructive",
};

export const commandTone = (c: ComputerCommand): SettingsStatusTone => tones[c.status];

export const commandLabel = (c: ComputerCommand): string => {
  if (c.status === "failed" && c.exit_code !== null) return `exit ${c.exit_code}`;
  if (c.status === "timed_out") return "timed out";
  return c.status;
};

export const commandLive = (c: ComputerCommand) => c.status === "queued" || c.status === "running";

export const commandOutputSource = (c: ComputerCommand): ContainerLogSource => ({
  path: c.machine_id ? `/runners/commands/${c.id}/output` : `/pairing/commands/${c.id}/output`,
  name: "command",
  forbidden: c.machine_id ? "You don't hold Run commands on this machine." : "This command ran on a computer that isn't yours.",
});
