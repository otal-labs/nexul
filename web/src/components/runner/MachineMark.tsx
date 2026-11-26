import { ServerIcon, ServerOffIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import type { GroupTone } from "@/utils/RunnerFleetUtility";

// A band's mark on the name's first line: a machine is a server glyph, not a dot, so the dots counted in the header are runners alone; it takes the warning hue only while some runners are down.
export const MachineMark = ({ tone, none = false }: { tone: GroupTone; none?: boolean }) => {
  const Icon = none ? ServerOffIcon : ServerIcon;
  return (
    <span className="flex h-5 shrink-0 items-center">
      <Icon aria-hidden className={cn("size-4 text-muted-foreground", tone === "partial" && "text-warning")} />
    </span>
  );
};
