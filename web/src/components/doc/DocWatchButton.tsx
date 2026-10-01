import { Eye, EyeOff } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { DocWatchersSection } from "@/components/doc/DocWatchersSection";
import { useFetchDocWatchers } from "@/hooks/DocHooks";
import { cn } from "@/lib/utils";

interface DocWatchButtonProps {
  docId: string;
}

// A doc's watchers are who its edits notify (ADR 0101); the eye is open while the viewer is one of them.
export const DocWatchButton = ({ docId }: DocWatchButtonProps) => {
  const { data } = useFetchDocWatchers(docId);
  const watching = data?.watching === true;
  const count = data?.watchers.length ?? 0;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className={cn("h-7 gap-1.5 px-2 text-muted-foreground hover:text-foreground", watching && "text-foreground")}
          aria-label={`Watchers: ${count}`}
        >
          {watching && <Eye className="size-4" aria-hidden />}
          {!watching && <EyeOff className="size-4" aria-hidden />}
          <span className="font-mono text-xs tabular-nums">{count}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-0">
        <DocWatchersSection docId={docId} />
      </PopoverContent>
    </Popover>
  );
};
