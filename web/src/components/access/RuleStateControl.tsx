import { Check, Lock, Slash, X } from "lucide-react";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { RuleState } from "@/models/Permission";
import { cn } from "@/lib/utils";

export const lockedAllowReason = "You can't allow a permission you don't hold";
export const lockedLiftReason = "You can't lift a Deny on a permission you don't hold";

const options = [
  { value: RuleState.Allow, word: "Allow", Icon: Check, on: "data-[state=on]:bg-success/15 data-[state=on]:text-success" },
  { value: RuleState.Fallback, word: "Fallback", Icon: Slash, on: "data-[state=on]:bg-accent data-[state=on]:text-foreground" },
  { value: RuleState.Deny, word: "Deny", Icon: X, on: "data-[state=on]:bg-destructive/15 data-[state=on]:text-destructive" },
];

interface RuleStateControlProps {
  label: string;
  value: RuleState;
  onChange: (value: RuleState) => void;
  // Why a state is closed to the giver, keyed by state: they don't hold the permission (ADR 0088).
  locked: Partial<Record<RuleState, string>>;
}

interface StateItemProps {
  state: RuleState;
  word: string;
  Icon: typeof Check;
  on: string;
  reason?: string | undefined;
}

// The tooltip hangs on a wrapper: on the button it would overwrite the on state, and a disabled button gets no hover.
const StateItem = ({ state, word, Icon, on, reason }: StateItemProps) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <span className="inline-flex">
        <ToggleGroupItem
          value={state}
          disabled={!!reason}
          aria-label={reason ? `${word}: ${reason}` : word}
          className={cn("w-7 rounded-[5px] px-0 text-muted-foreground hover:bg-accent hover:text-foreground", on)}
        >
          {reason && <Lock className="size-3" aria-hidden />}
          {!reason && <Icon className="size-3.5" aria-hidden />}
        </ToggleGroupItem>
      </span>
    </TooltipTrigger>
    <TooltipContent>{reason ?? word}</TooltipContent>
  </Tooltip>
);

// Three icon buttons as one radio group: the one that is on is filled in its own hue, a closed one shows a lock.
export const RuleStateControl = ({ label, value, onChange, locked }: RuleStateControlProps) => (
  <ToggleGroup
    type="single"
    size="xs"
    spacing={1}
    value={value}
    aria-label={label}
    onValueChange={(v) => v && onChange(v as RuleState)}
    className="shrink-0 gap-0.5 rounded-md bg-surface-2 p-0.5 ring-1 ring-border ring-inset"
  >
    {options.map((o) => (
      <StateItem key={o.value} state={o.value} word={o.word} Icon={o.Icon} on={o.on} reason={locked[o.value]} />
    ))}
  </ToggleGroup>
);
