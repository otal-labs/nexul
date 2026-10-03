import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { TEMPLATE } from "@/components/memory/prototype/InterviewPrototypeData";
import { useInterviewPrototypeStore, type ProtoSnapshot } from "@/components/memory/prototype/InterviewPrototypeStore";
import type { TrailState } from "@/models/Trail";
import { cn } from "@/lib/utils";

export type StepTone = "done" | "person" | "agent" | "stale" | "upcoming";

export interface RunStep {
  label: string;
  detail: string;
  tone: StepTone;
}

type RunView = Omit<ProtoSnapshot, "drafts" | "collapsed">;

const LABELS = ["Answering", "Agent reading", "Agent asking", "Memory ready"];

const stageOf = (s: RunView): number => {
  if (s.run === "reading") return 1;
  if (s.run === "asking") return 2;
  if (s.run === "writing") return 3;
  if (s.run === "ready" && s.memory === "fresh") return 4;
  return 0;
};

const answeringDetail = (s: RunView): string => {
  const answered = TEMPLATE.filter((q) => s.answers[q.id] !== undefined).length;
  const skipped = TEMPLATE.filter((q) => s.skipped.includes(q.id)).length;
  if (skipped === 0) return `${answered} of ${TEMPLATE.length} answered`;
  return `${answered} answered · ${skipped} skipped`;
};

const memoryDetail = (s: RunView): string => {
  if (s.run === "writing") return "Writing from your answers…";
  if (s.memory === "legacy") return "From an earlier interview, 14 Aug";
  if (s.changed.length > 0) return `${s.changed.length} answer changed`;
  if (s.memory === "fresh") return "Written 2 min ago";
  return "";
};

const detailOf = (s: RunView, i: number, stage: number): string => {
  if (i === 0) return answeringDetail(s);
  if (i === 1) return stage === 1 ? "Your answers and the checkout" : "";
  if (i === 2 && stage === 2) return `Follow-up ${s.answers.f1 ? 2 : 1} of ${s.followUps}`;
  if (i === 2 && stage > 2) return `${s.followUps} follow-ups answered`;
  if (i === 3) return memoryDetail(s);
  return "";
};

const toneOf = (s: RunView, i: number, stage: number): StepTone => {
  if (i === 3 && s.memory !== "none" && s.changed.length > 0) return "stale";
  if (i === 3 && s.memory !== "none" && s.run !== "writing") return "done";
  if (i < stage) return "done";
  if (i > stage) return "upcoming";
  return i === 0 ? "person" : "agent";
};

export const useRunSteps = (): { steps: RunStep[]; current: RunStep } => {
  const s = useInterviewPrototypeStore(
    useShallow((st) => ({ answers: st.answers, skipped: st.skipped, openId: st.openId, followUps: st.followUps, run: st.run, memory: st.memory, changed: st.changed, memoryDot: st.memoryDot })),
  );
  const stage = stageOf(s);
  const steps = LABELS.map((label, i) => ({ label, detail: detailOf(s, i, stage), tone: toneOf(s, i, stage) }));
  const current = s.memory !== "none" && stage === 0 ? steps[3]! : steps[Math.min(stage, 3)]!;
  return { steps, current };
};

const DOT: Record<StepTone, string> = {
  done: "bg-success",
  person: "bg-foreground",
  agent: "bg-warning animate-[status-pulse_2.4s_ease-standard_infinite]",
  stale: "bg-warning",
  upcoming: "border border-muted-foreground/50",
};

export const RunDot = ({ tone, className }: { tone: StepTone; className?: string }) => (
  <span aria-hidden className={cn("inline-block size-2 shrink-0 rounded-full", DOT[tone], className)} />
);

const trailStateOf = (step: RunStep): TrailState | null => {
  if (step.tone === "done") return "done";
  if (step.tone !== "agent") return null;
  return step.label === "Agent asking" ? "waiting" : "running";
};

// The run state as one line: the dot, the step, and its detail; trailIcons swaps the dot for the trail rows' state icon.
export const RunLine = ({ className, trailIcons = false }: { className?: string; trailIcons?: boolean }) => {
  const { current } = useRunSteps();
  const trailState = trailIcons ? trailStateOf(current) : null;
  return (
    <p className={cn("flex min-w-0 items-center gap-2 text-sm", className)}>
      {trailState && <TrailStateIcon state={trailState} />}
      {!trailState && <RunDot tone={current.tone} />}
      <span className="shrink-0 font-medium">{current.label}</span>
      {current.detail && <span className="truncate text-muted-foreground">· {current.detail}</span>}
    </p>
  );
};

// Shown once a memory exists; primary with its full name when answers changed since it was written.
export const RegenerateButton = ({ className }: { className?: string }) => {
  const { memory, run, changed, regenerate } = useInterviewPrototypeStore(
    useShallow((s) => ({ memory: s.memory, run: s.run, changed: s.changed.length > 0, regenerate: s.regenerate })),
  );
  if (memory === "none") return null;
  return (
    <Button
      size="sm"
      variant={changed ? "default" : "outline"}
      loading={run === "writing"}
      disabled={run === "writing"}
      onClick={regenerate}
      className={className}
    >
      {changed ? "Regenerate the memory" : "Regenerate"}
    </Button>
  );
};
