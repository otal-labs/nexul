import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useOpenRow } from "@/hooks/useOpenRow";
import { cn } from "@/lib/utils";

export const newAutoPlayKey = (playId: string) => `new:${playId}`;

interface NewAutoPlayRowProps {
  playId: string;
  open: boolean;
  // Always drawn while the play has no auto plays; otherwise it takes no room until the play is hovered or holds focus, so Tab from the play reaches it.
  shown: boolean;
}

// The compact, muted last row of a play's branch; it unfolds a new auto play's editor in place.
export const NewAutoPlayRow = ({ playId, open, shown }: NewAutoPlayRowProps) => {
  const { toggle } = useOpenRow();
  return (
    <div
      className={cn(
        "relative flex h-7 items-center border-b border-border pl-12",
        !shown && !open && "hidden group-focus-within/play:flex group-hover/play:flex",
        open && currentRowFillClass,
      )}
    >
      {open && <CurrentRowEdge />}
      <ActionButton
        permission="autoplays:write"
        verb="add auto plays"
        type="button"
        variant="ghost"
        size="sm"
        aria-expanded={open}
        className="-ml-2.5 h-6 px-2.5 text-xs font-normal text-muted-foreground"
        onClick={() => toggle(newAutoPlayKey(playId))}
      >
        + Start it by itself when…
      </ActionButton>
    </div>
  );
};
