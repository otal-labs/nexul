import { cn } from "@/lib/utils";
import type { GroupTone } from "@/utils/RunnerFleetUtility";

// A band's state dot, the same grammar as a runner row's: green online, amber while some runners are down, a hollow ring offline.
export const MachineMark = ({ tone }: { tone: GroupTone }) => (
  <span
    aria-hidden
    className={cn(
      "size-2 shrink-0 self-center rounded-full",
      tone === "online" && "bg-success",
      tone === "partial" && "bg-warning-mark",
      tone === "offline" && "ring-[1.5px] ring-muted-foreground/60 ring-inset",
    )}
  />
);
