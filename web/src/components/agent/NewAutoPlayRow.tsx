import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useOpenRow } from "@/hooks/useOpenRow";
import { cn } from "@/lib/utils";

export const newAutoPlayKey = (playId: string) => `new:${playId}`;

// The compact, muted last row of a play's branch, its text on the auto plays' chevron; it unfolds a new auto play's editor in place.
export const NewAutoPlayRow = ({ playId, open }: { playId: string; open: boolean }) => {
  const { toggle } = useOpenRow();
  return (
    <div className={cn("relative flex h-6 items-center border-b border-border pl-12", open && currentRowFillClass)}>
      {open && <CurrentRowEdge />}
      <ActionButton
        permission="autoplays:write"
        verb="add auto plays"
        type="button"
        variant="ghost"
        size="sm"
        aria-expanded={open}
        className="h-6 px-0 text-xs font-normal text-muted-foreground hover:bg-transparent hover:text-foreground"
        onClick={() => toggle(newAutoPlayKey(playId))}
      >
        + Add auto play
      </ActionButton>
    </div>
  );
};
