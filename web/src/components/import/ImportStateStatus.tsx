import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { ImportState } from "@/models/Import";

const tone: Record<ImportState, SettingsStatusTone> = {
  scanning: "info",
  ready: "warning",
  importing: "info",
  completed: "success",
  failed: "destructive",
  cancelled: "muted",
};

const label: Record<ImportState, string> = {
  scanning: "Scanning",
  ready: "Ready to import",
  importing: "Importing",
  completed: "Completed",
  failed: "Failed",
  cancelled: "Cancelled",
};

interface ImportStateStatusProps {
  state: ImportState;
  detail?: string;
}

export const ImportStateStatus = ({ state, detail }: ImportStateStatusProps) => (
  <SettingsStatus tone={tone[state]} detail={detail}>
    {label[state]}
  </SettingsStatus>
);
