import { WizardRailNode } from "@/components/wizard/WizardRailNode";
import { RAIL_MARK_WORD, type WizardRailMark } from "@/models/WizardRail";
import { cn } from "@/lib/utils";

interface WizardRailRowProps {
  label: string;
  mark: WizardRailMark;
  // The step's place among the rail's top-level steps, "04", or "5.2" for a sub-step.
  ordinal: string;
  total: number;
  // Only a step its flow can open is a button.
  onOpen?: (() => void) | undefined;
  // A sub-step sits one indent in, its connector on its own nodes.
  nested?: boolean;
  nextNested?: boolean;
  // A sub-step with a top-level step after it carries the main connector past itself.
  spine?: boolean;
  last: boolean;
}

const metaLine = (mark: WizardRailMark, ordinal: string, total: number): string => {
  if (mark === "current") return `Step ${ordinal.replace(/^0/, "")} of ${total}`;
  if (mark === "left") return ordinal;
  return RAIL_MARK_WORD[mark];
};

// One step of any wizard's rail: its label over a mono line saying where it stands; the connector runs node edge to node edge.
export const WizardRailRow = ({ label, mark, ordinal, total, onOpen, nested = false, nextNested = false, spine = false, last }: WizardRailRowProps) => {
  const current = mark === "current";
  const Tag = onOpen ? "button" : "div";

  return (
    <li className={cn("relative", nested && "pl-8")} data-state={mark === "left" ? "unvisited" : mark}>
      {!last && (!nested || spine) && (
        <span aria-hidden className={cn("absolute -bottom-2 left-[calc(0.5rem+9.5px)] w-px bg-border", nested ? "top-0" : "top-7")} />
      )}
      {!last && nested && nextNested && <span aria-hidden className="absolute top-7 -bottom-2 left-[calc(2.5rem+9.5px)] w-px bg-border" />}
      <Tag
        {...(onOpen && { type: "button" as const, onClick: onOpen })}
        aria-current={current ? "step" : undefined}
        className={cn(
          "group inset-focus relative flex w-full items-start gap-3 rounded-md px-2 py-2 text-left transition-colors duration-150 ease-standard",
          onOpen && "hover:bg-accent",
        )}
      >
        <WizardRailNode mark={mark} />
        <span className="flex min-w-0 flex-col">
          <span
            className={cn(
              "text-sm leading-5",
              current && "font-medium text-foreground",
              !current && "text-muted-foreground",
              !current && onOpen && "group-hover:text-foreground",
            )}
          >
            {label}
          </span>
          <span className="font-mono text-[11px] leading-4 tracking-[0.06em] text-muted-foreground uppercase tabular-nums">
            <span className="sr-only">, </span>
            {metaLine(mark, ordinal, total)}
          </span>
        </span>
      </Tag>
    </li>
  );
};
