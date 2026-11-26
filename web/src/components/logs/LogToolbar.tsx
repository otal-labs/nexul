import type { ReactNode } from "react";
import { CopyIcon, DownloadIcon, PauseIcon, PlayIcon } from "lucide-react";

import { LogStatusMarker } from "@/components/logs/LogStatusMarker";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  // False with nothing received and nothing coming, where All and Errors would both show an empty block.
  filterable: boolean;
  // What picks the log (a stack's service), led on the same row so the page keeps one tab row.
  leading?: ReactNode;
}

interface ToolButtonProps {
  label: string;
  icon: typeof CopyIcon;
  onClick: () => void;
  disabled: boolean;
  pressed?: boolean;
}

// Pause, Copy and Download are well-known tools, so they are quiet icons after the status marker; the row then fits one
// line, and the filter is its one boxed control.
const ToolButton = ({ label, icon: Icon, onClick, disabled, pressed }: ToolButtonProps) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <Button type="button" variant="ghost" size="icon" aria-label={label} aria-pressed={pressed} onClick={onClick} disabled={disabled} className="size-8 text-muted-foreground hover:text-foreground">
        <Icon aria-hidden />
      </Button>
    </TooltipTrigger>
    <TooltipContent>{label}</TooltipContent>
  </Tooltip>
);

export const LogToolbar = ({ filter, onFilterChange, status, reason, paused, onPausedChange, onCopy, onDownload, empty, filterable, leading }: LogToolbarProps) => (
  <div className="flex flex-wrap items-center gap-2">
    {leading}
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={filter}
      onValueChange={(value) => value && onFilterChange(value as LogFilter)}
      disabled={!filterable}
      aria-label="Show"
    >
      <ToggleGroupItem value="all">All</ToggleGroupItem>
      <ToggleGroupItem value="errors">Errors</ToggleGroupItem>
    </ToggleGroup>
    <LogStatusMarker status={status} reason={reason} paused={paused} />
    <div className="flex items-center border-l border-border pl-1">
      <ToolButton
        label={paused ? "Resume" : "Pause"}
        icon={paused ? PlayIcon : PauseIcon}
        onClick={() => onPausedChange(!paused)}
        disabled={status !== "live" && !paused}
        pressed={paused}
      />
      <ToolButton label="Copy" icon={CopyIcon} onClick={onCopy} disabled={empty} />
      <ToolButton label="Download" icon={DownloadIcon} onClick={onDownload} disabled={empty} />
    </div>
  </div>
);
