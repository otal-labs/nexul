import { treeRowClass } from "@/components/agent/treeRow";
import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useOpenRow } from "@/hooks/useOpenRow";
import { cn } from "@/lib/utils";

export const newAutoPlayKey = (playId: string) => `new:${playId}`;

// The muted last row of a play's branch; it unfolds a new auto play's editor in place.
export const NewAutoPlayRow = ({ playId, open }: { playId: string; open: boolean }) => {
  const { toggle } = useOpenRow();
  return (
    <div className={cn(treeRowClass, "py-0.5 pl-12", open && currentRowFillClass)}>
      {open && <CurrentRowEdge />}
      <ActionButton
        permission="autoplays:write"
        verb="add auto plays"
        type="button"
        variant="ghost"
        size="sm"
        aria-expanded={open}
        className="col-span-2 -ml-2.5 justify-start font-normal text-muted-foreground sm:col-span-4"
        onClick={() => toggle(newAutoPlayKey(playId))}
      >
        + Start it by itself when…
      </ActionButton>
    </div>
  );
};
