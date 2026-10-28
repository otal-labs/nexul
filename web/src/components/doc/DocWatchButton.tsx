import { Eye } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DocWatchersSection } from "@/components/doc/DocWatchersSection";
import { useFetchDocWatchers } from "@/hooks/DocHooks";
import { cn } from "@/lib/utils";

interface DocWatchButtonProps {
  docId: string;
}

// A doc's watchers are who its edits notify (ADR 0101); the eye is ink while the viewer is one of them.
export const DocWatchButton = ({ docId }: DocWatchButtonProps) => {
  const { data } = useFetchDocWatchers(docId);
  const watching = data?.watching === true;
  const count = data?.watchers.length ?? 0;
  const label = `${watching ? "Watching" : "Not watching"}, ${count} ${count === 1 ? "watcher" : "watchers"}`;

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className={cn("h-7 gap-1.5 px-2 text-muted-foreground hover:text-foreground", watching && "text-foreground")}
              aria-label={label}
            >
              <Eye className="size-4" aria-hidden />
              <span className="font-mono text-xs tabular-nums">{count}</span>
            </Button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
      <PopoverContent align="end" className="w-64 p-0">
        <DocWatchersSection docId={docId} />
      </PopoverContent>
    </Popover>
  );
};
