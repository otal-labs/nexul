import { Check, Circle, CircleDashed } from "lucide-react";

import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { gapPhrase, gapText, people, skipNoun, skippedText, stepName, tally, type ProtoPerson, type ProtoStep } from "./data";
import { RowShell, showsProgress } from "./shared";

interface VariantProps {
  onOpen: (id: string) => void;
}

// 1. Gap line: one status line under the meta line, naming what is left; nothing once every step is done.
export const GapLine = ({ person }: { person: ProtoPerson }) => {
  const { done, total, left, skipped } = tally(person.steps);
  if (!showsProgress(person)) return null;
  if (left.length === 0 && skipped.length === 0) return null;
  return (
    <span className="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
      {left.length > 0 && (
        <SettingsStatus tone="warning" detail={gapText(left)}>
          <span className="font-mono font-normal tabular-nums">{done} of {total}</span> set up
        </SettingsStatus>
      )}
      {left.length === 0 && (
        <SettingsStatus tone="muted">
          <span className="font-mono font-normal tabular-nums">{done} of {total}</span> set up
        </SettingsStatus>
      )}
      {skipped.length > 0 && <span className="text-xs text-muted-foreground">· {skippedText(skipped)}</span>}
    </span>
  );
};

export const GapLineVariant = ({ onOpen }: VariantProps) => (
  <ul aria-label="Team" className="divide-y divide-border overflow-hidden rounded-md border">
    {people.map((person) => (
      <RowShell key={person.id} person={person} onOpen={onOpen} extraLine={<GapLine person={person} />} />
    ))}
  </ul>
);

// 2. Progress column: a segment per step and a mono count in a fixed column, the gap named on hover.
const segmentClass: Record<ProtoStep["state"], string> = {
  done: "bg-brand",
  left: "bg-foreground/25",
  skipped: "border border-dashed border-muted-foreground/60",
};

const ProgressColumn = ({ person }: { person: ProtoPerson }) => {
  const { done, total, left, skipped } = tally(person.steps);
  const quiet = !showsProgress(person) || (left.length === 0 && skipped.length === 0);
  const hint = [left.length > 0 && left.map((step) => gapPhrase[step.key]).join(", "), skipped.length > 0 && skippedText(skipped)]
    .filter(Boolean)
    .join(" · ");
  return (
    <span className="flex w-24 shrink-0 flex-col items-start gap-1.5 self-start pt-1">
      {!quiet && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex flex-col gap-1.5" aria-label={`${done} of ${total} welcome steps done: ${hint}`}>
              <span className="flex gap-0.5">
                {person.steps.map((step) => (
                  <span key={step.key} className={cn("h-1.5 w-3 rounded-[2px]", segmentClass[step.state])} />
                ))}
              </span>
              <span className="font-mono text-xs text-muted-foreground tabular-nums">
                {done}/{total} set up
              </span>
            </span>
          </TooltipTrigger>
          <TooltipContent>{hint}</TooltipContent>
        </Tooltip>
      )}
    </span>
  );
};

export const ColumnVariant = ({ onOpen }: VariantProps) => (
  <ul aria-label="Team" className="divide-y divide-border overflow-hidden rounded-md border">
    {people.map((person) => (
      <RowShell key={person.id} person={person} onOpen={onOpen} column={<ProgressColumn person={person} />} />
    ))}
  </ul>
);

// 3. Step strip: every step on the row as an icon and its name, so the whole wizard reads without opening it.
const StepMark = ({ step }: { step: ProtoStep }) => (
  <span className={cn("inline-flex items-center gap-1 text-xs", step.state === "left" ? "text-foreground" : "text-muted-foreground")}>
    {step.state === "done" && <Check aria-hidden className="size-3.5 text-success" />}
    {step.state === "left" && <Circle aria-hidden className="size-3 text-warning" />}
    {step.state === "skipped" && <CircleDashed aria-hidden className="size-3" />}
    <span>{step.state === "skipped" ? `${skipNoun[step.key]} skipped` : stepName[step.key]}</span>
  </span>
);

const StepStrip = ({ person }: { person: ProtoPerson }) => {
  const { left, skipped } = tally(person.steps);
  if (!showsProgress(person)) return null;
  if (left.length === 0 && skipped.length === 0) return null;
  return (
    <span className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1">
      {person.steps.map((step) => (
        <StepMark key={step.key} step={step} />
      ))}
    </span>
  );
};

export const StripVariant = ({ onOpen }: VariantProps) => (
  <ul aria-label="Team" className="divide-y divide-border overflow-hidden rounded-md border">
    {people.map((person) => (
      <RowShell key={person.id} person={person} onOpen={onOpen} extraLine={<StepStrip person={person} />} />
    ))}
  </ul>
);
