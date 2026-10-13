import { EmptyRow } from "@/components/EmptyRow";
import { microheaderClass } from "@/components/Microheader";
import { menuItemClass } from "@/components/MenuItem";
import { TargetMark } from "@/components/access/TargetMark";
import type { PanelTarget } from "@/hooks/usePanelTargets";
import { cn } from "@/lib/utils";

interface TargetChoiceProps {
  target: PanelTarget;
  // Set when the target can't be listed here yet: it isn't in the doc's project.
  blocked?: string | undefined;
  onPick: () => void;
  // Opens the project's list on the target, for a viewer who manages the project.
  onBlocked?: (() => void) | undefined;
}

const TargetChoice = ({ target, blocked, onPick, onBlocked }: TargetChoiceProps) => (
  <>
    {!blocked && (
      <button type="button" onClick={onPick} className={cn(menuItemClass, "w-full")}>
        <TargetMark target={target} className="size-5" />
        <span className="min-w-0 flex-1 truncate">{target.name}</span>
      </button>
    )}
    {blocked && (
      <div aria-disabled="true" className={cn(menuItemClass, "w-full cursor-default hover:bg-transparent")}>
        <TargetMark target={target} className="size-5 opacity-50" />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-muted-foreground">{target.name}</span>
          {onBlocked && (
            <button type="button" onClick={onBlocked} className="block text-left text-xs text-muted-foreground underline decoration-border underline-offset-2 hover:text-foreground">
              {blocked}
            </button>
          )}
          {!onBlocked && <span className="block text-xs text-muted-foreground">{blocked}</span>}
        </span>
      </div>
    )}
  </>
);

interface TargetChoicesProps {
  roles: PanelTarget[];
  people: PanelTarget[];
  blockedFor: (target: PanelTarget) => string | undefined;
  onPick: (target: PanelTarget) => void;
  onBlocked?: ((target: PanelTarget) => void) | undefined;
}

// The add popover's body in the menu anatomy: the roles and people not listed yet, under their microheaders.
export const TargetChoices = ({ roles, people, blockedFor, onPick, onBlocked }: TargetChoicesProps) => (
  <>
    {roles.length > 0 && <p className={cn(microheaderClass, "px-2 pt-2 pb-1")}>Roles</p>}
    {roles.map((t) => (
      <TargetChoice key={t.key} target={t} blocked={blockedFor(t)} onPick={() => onPick(t)} onBlocked={onBlocked && (() => onBlocked(t))} />
    ))}
    {people.length > 0 && <p className={cn(microheaderClass, "px-2 pt-2 pb-1")}>People</p>}
    {people.map((t) => (
      <TargetChoice key={t.key} target={t} blocked={blockedFor(t)} onPick={() => onPick(t)} onBlocked={onBlocked && (() => onBlocked(t))} />
    ))}
    {roles.length + people.length === 0 && <EmptyRow className="px-2 py-2">Every role and person is listed.</EmptyRow>}
  </>
);
