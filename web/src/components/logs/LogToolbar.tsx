import { CopyIcon, DownloadIcon, PauseIcon, PlayIcon } from "lucide-react";

import { LogStatusMarker } from "@/components/logs/LogStatusMarker";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { LogFilter, LogStatus } from "@/models/ContainerLog";

interface LogToolbarProps {
  filter: LogFilter;
  onFilterChange: (filter: LogFilter) => void;
  status: LogStatus;
  reason: string | undefined;
  paused: boolean;
  onPausedChange: (paused: boolean) => void;
  onCopy: () => void;
  onDownload: () => void;
  empty: boolean;
}

export const LogToolbar = ({ filter, onFilterChange, status, reason, paused, onPausedChange, onCopy, onDownload, empty }: LogToolbarProps) => (
  <div className="flex flex-wrap items-center gap-2">
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={filter}
      onValueChange={(value) => value && onFilterChange(value as LogFilter)}
      aria-label="Show"
    >
      <ToggleGroupItem value="all">All</ToggleGroupItem>
      <ToggleGroupItem value="errors">Errors</ToggleGroupItem>
    </ToggleGroup>
    <Button type="button" variant="outline" size="sm" onClick={() => onPausedChange(!paused)}>
      {paused && <PlayIcon aria-hidden />}
      {!paused && <PauseIcon aria-hidden />}
      {paused ? "Resume" : "Pause"}
    </Button>
    <Button type="button" variant="outline" size="sm" onClick={onCopy} disabled={empty}>
      <CopyIcon aria-hidden /> Copy
    </Button>
    <Button type="button" variant="outline" size="sm" onClick={onDownload} disabled={empty}>
      <DownloadIcon aria-hidden /> Download
    </Button>
    <LogStatusMarker status={status} reason={reason} paused={paused} />
  </div>
);
