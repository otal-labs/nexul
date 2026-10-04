import { Sparkles } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { pendingCount, questionsUpTo, useClarifyDev, useClarifyPrototypeStore, waitingLabel } from "@/components/doc/prototype/ClarifyPrototypeStore";
import type { TrailState } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface StatusLine {
  icon: TrailState | null;
  label: string;
  detail: string;
}

const THANKS = "Thanks, nothing is waiting on you.";

const useStatusLine = (): StatusLine => {
  const dev = useClarifyDev();
  const s = useClarifyPrototypeStore(useShallow((st) => ({ ...st })));
  const pending = pendingCount(s);
  const answered = questionsUpTo(s.rounds).filter((q) => s.answers[q.id] !== undefined).length;
  const rounds = `${s.rounds} ${s.rounds === 1 ? "round" : "rounds"}`;
  if (s.phase === "running" && dev) return { icon: "running", label: `Round ${s.rounds + 1} running`, detail: "The doc is locked until it ends" };
  if (s.phase === "running") return { icon: null, label: "More questions are on the way", detail: "" };
  if (s.phase === "open" && pending > 0) return { icon: "waiting", label: waitingLabel(pending), detail: `Round ${s.rounds}` };
  if (s.phase === "open" && dev) return { icon: "done", label: `Round ${s.rounds} answered`, detail: "" };
  if (s.phase === "nogaps" && dev) return { icon: "done", label: "No gaps left", detail: "The doc now holds every answer" };
  if (s.phase === "closed" && dev) return { icon: "done", label: "Closed", detail: `${rounds} · ${answered} answered` };
  if (s.phase === "closed") return { icon: "done", label: "All answered", detail: rounds };
  return { icon: "done", label: "All answered", detail: THANKS };
};

// The clarification's state as one line: the trail icon, what is happening, a muted detail.
export const ClarifyPrototypeStatus = ({ className }: { className?: string }) => {
  const { icon, label, detail } = useStatusLine();
  return (
    <span className={cn("flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-sm", className)}>
      {icon && <TrailStateIcon state={icon} />}
      <span className={cn(icon ? "font-medium" : "text-muted-foreground")}>{label}</span>
      {detail !== "" && <span className="text-muted-foreground">· {detail}</span>}
    </span>
  );
};

const ClarifyButton = ({ primary }: { primary: boolean }) => {
  const clarify = useClarifyPrototypeStore((s) => s.clarify);
  return (
    <Button size="sm" variant={primary ? "default" : "outline"} onClick={clarify}>
      <Sparkles className="size-3.5" aria-hidden />
      Clarify via AI
    </Button>
  );
};

// What someone who can run plays may do next; the client sees none of it.
export const ClarifyPrototypeActions = () => {
  const dev = useClarifyDev();
  const s = useClarifyPrototypeStore(useShallow((st) => ({ ...st })));
  const pending = pendingCount(s);
  if (!dev) return null;
  return (
    <span className="flex shrink-0 items-center gap-2">
      {s.phase === "open" && <ClarifyButton primary={pending === 0} />}
      {s.phase === "nogaps" && (
        <Button size="sm" onClick={s.close}>
          Close
        </Button>
      )}
      {s.phase === "closed" && <ClarifyButton primary={false} />}
      {s.phase === "closed" && (
        <Button size="sm">
          <Sparkles className="size-3.5" aria-hidden />
          To tickets via AI
        </Button>
      )}
    </span>
  );
};
