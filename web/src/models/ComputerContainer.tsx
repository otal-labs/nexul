import type { SettingsStatusTone } from "@/components/settings/SettingsStatus";

// Mirrors pairing.Container: one Docker container on a person's computer, which Nexul only reads.
export interface ComputerContainer {
  name: string;
  image: string;
  status: string;
}

// Docker's own states; an unknown one stays muted.
const STATUS_TONE: Record<string, SettingsStatusTone> = {
  running: "success",
  restarting: "warning",
  paused: "info",
  dead: "destructive",
};

export const containerStatusTone = (status: string): SettingsStatusTone => STATUS_TONE[status] ?? "muted";
