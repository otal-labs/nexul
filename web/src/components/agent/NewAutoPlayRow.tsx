import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useOpenRow } from "@/hooks/useOpenRow";
import { cn } from "@/lib/utils";

export const newAutoPlayKey = (playId: string) => `new:${playId}`;

interface NewAutoPlayRowProps {
  playId: string;
  open: boolean;
  // Always drawn while the play has no auto plays; otherwise it shows with the play row hovered, focused or open, and stays in the tab order.
  shown: boolean;
}

// The compact, muted last row of a play's branch; it unfolds a new auto play's editor in place.
export const NewAutoPlayRow = ({ playId, open, shown }: NewAutoPlayRowProps) => {
  const { toggle } = useOpenRow();
  return (
    <div className={cn("relative flex h-7 items-center border-b border-border pl-12", open && currentRowFillClass)}>
      {open && <CurrentRowEdge />}
      <ActionButton
        permission="autoplays:write"
        verb="add auto plays"
        type="button"
        variant="ghost"
        size="sm"
        aria-expanded={open}
        className={cn(
          "-ml-2.5 h-6 px-2.5 text-xs font-normal text-muted-foreground transition-opacity duration-150 ease-standard focus-visible:opacity-100 group-focus-within/play:opacity-100 group-hover/play:opacity-100",
          !shown && !open && "opacity-0",
        )}
        onClick={() => toggle(newAutoPlayKey(playId))}
      >
        + Start it by itself when…
      </ActionButton>
    </div>
  );
};
