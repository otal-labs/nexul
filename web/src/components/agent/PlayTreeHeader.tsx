import { treeGridClass } from "@/components/agent/treeRow";
import { cn } from "@/lib/utils";

export const PlayTreeHeader = () => (
  <div className={cn(treeGridClass, "border-b border-border px-4 py-2 text-xs text-muted-foreground")}>
    <span>Play, and when it starts by itself</span>
    <span className="hidden text-right sm:block">Runs · 7d</span>
    <span className="hidden text-right lg:block">Last run</span>
    <span className="text-right">On</span>
  </div>
);
