import { cn } from "@/lib/utils";
import type { GroupTone } from "@/utils/RunnerFleetUtility";

// A band's dot, on the runner dots' x and the band name's first line.
export const GroupDot = ({ tone }: { tone: GroupTone }) => (
  <span className="flex h-5 shrink-0 items-center" title={tone}>
    <span
      aria-hidden
      className={cn(
        "size-2 rounded-full",
        tone === "online" && "bg-success",
        tone === "partial" && "bg-warning",
        tone === "offline" && "ring-[1.5px] ring-muted-foreground/60 ring-inset",
      )}
    />
  </span>
);
