import { ScrollTextIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { containerStatusTone, type ComputerContainer } from "@/models/ComputerContainer";

interface ComputerContainerRowProps {
  container: ComputerContainer;
  onLogs: (name: string) => void;
}

export const ComputerContainerRow = ({ container, onLogs }: ComputerContainerRowProps) => (
  <li className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 py-1">
    <span className="min-w-0 font-mono break-all text-foreground">{container.name}</span>
    <SettingsStatus tone={containerStatusTone(container.status)}>{container.status}</SettingsStatus>
    <span className="min-w-0 flex-1 truncate font-mono text-muted-foreground">{container.image}</span>
    <Button type="button" variant="ghost" size="sm" aria-label={`Logs of ${container.name}`} onClick={() => onLogs(container.name)}>
      <ScrollTextIcon aria-hidden /> Logs
    </Button>
  </li>
);
