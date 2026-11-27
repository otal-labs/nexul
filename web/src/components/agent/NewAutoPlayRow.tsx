import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { inlineActionClass } from "@/components/autoplay/ComposerControls";
import { ActionButton } from "@/components/action/ActionButton";
import { useOpenRow } from "@/hooks/useOpenRow";
import { cn } from "@/lib/utils";

export const newAutoPlayKey = (playId: string) => `new:${playId}`;

interface NewAutoPlayRowProps {
  playId: string;
  open: boolean;
  // Drawn while the play is open or has no auto plays; otherwise it takes no room until the play is hovered or holds focus.
  shown: boolean;
}

// The compact, muted last row of a play's branch, its text on the auto plays' sentences; it unfolds a new auto play's editor in place.
// Folded to no height rather than display none, so Tab from the play still reaches it and the focus reveals it.
export const NewAutoPlayRow = ({ playId, open, shown }: NewAutoPlayRowProps) => {
  const { toggle } = useOpenRow();
  return (
    <div
      className={cn(
        "relative flex items-center overflow-hidden pl-[4.375rem]",
        shown || open ? "h-6 border-b border-border" : "h-0 group-focus-within/play:h-6 group-focus-within/play:border-b group-hover/play:h-6 group-hover/play:border-b group-hover/play:border-border group-focus-within/play:border-border",
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
        className={cn(inlineActionClass, "h-6 px-0 font-normal hover:bg-transparent")}
        onClick={() => toggle(newAutoPlayKey(playId))}
      >
        + Add auto play
      </ActionButton>
    </div>
  );
};
