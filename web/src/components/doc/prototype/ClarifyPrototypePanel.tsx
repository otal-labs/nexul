import { ClarifyPrototypeRound } from "@/components/doc/prototype/ClarifyPrototypeRound";
import { ClarifyPrototypeActions, ClarifyPrototypeStatus } from "@/components/doc/prototype/ClarifyPrototypeStatus";
import { pendingCount, useClarifyDev, useClarifyPrototypeStore, waitingLabel } from "@/components/doc/prototype/ClarifyPrototypeStore";
import { cn } from "@/lib/utils";

export const PANEL_ID = "clarify-questions";

// Every round so far, oldest first, the way the interview appends its follow-ups.
export const ClarifyPrototypeRounds = ({ className }: { className?: string }) => {
  const rounds = useClarifyPrototypeStore((s) => s.rounds);
  return (
    <div className={cn("space-y-6", className)}>
      {Array.from({ length: rounds }, (_, i) => (
        <ClarifyPrototypeRound key={i + 1} n={i + 1} />
      ))}
    </div>
  );
};

// The Questions panel: its title with the next action, the state line, then the rounds.
export const ClarifyPrototypePanel = ({ className }: { className?: string }) => (
  <section id={PANEL_ID} aria-label="Questions" className={cn("min-w-0 scroll-mt-6", className)}>
    <header className="flex flex-wrap items-end justify-between gap-x-3 gap-y-3 border-b border-border pb-3">
      <div className="min-w-0 flex-1 basis-56 space-y-1">
        <h2 className="text-base font-semibold tracking-tight">Questions</h2>
        <ClarifyPrototypeStatus />
      </div>
      <ClarifyPrototypeActions />
    </header>
    <ClarifyPrototypeRounds className="mt-5" />
  </section>
);

// The header's "waiting on you" marker, for people who answer rather than run rounds.
export const ClarifyPrototypeMarker = () => {
  const dev = useClarifyDev();
  const pending = useClarifyPrototypeStore((s) => pendingCount(s));
  if (dev || pending === 0) return null;
  return (
    <a
      href={`#${PANEL_ID}`}
      className="flex items-center gap-1.5 rounded-md px-2 py-1 font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground"
    >
      <span className="size-1.5 rounded-full bg-info" aria-hidden />
      {waitingLabel(pending)}
    </a>
  );
};
