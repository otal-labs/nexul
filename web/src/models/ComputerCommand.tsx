import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";
import type { ContainerLogSource } from "@/models/ContainerLog";

export type CommandStatus = "queued" | "running" | "succeeded" | "failed" | "timed_out" | "cancelled" | "refused";

// Mirrors pairing.ShellJob: one command run on a computer through its runner, kept with who ran it.
export interface ComputerCommand {
  id: string;
  computer_id: string;
  owner_id: string;
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
  path: `/pairing/commands/${c.id}/output`,
  name: "command",
  forbidden: "This command ran on a computer that isn't yours.",
});
