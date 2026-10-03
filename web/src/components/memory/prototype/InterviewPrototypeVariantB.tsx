import { NotebookPen } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { EmptyState } from "@/components/EmptyState";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewPrototypeChecklist } from "@/components/memory/prototype/InterviewPrototypeChecklist";
import { InterviewPrototypeMemory } from "@/components/memory/prototype/InterviewPrototypeMemory";
import { RegenerateButton, RunLine } from "@/components/memory/prototype/InterviewPrototypeRun";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";

// B: two columns, the right one is the memory with the run state as its header line; below 1024px of page it drops under.
export const InterviewPrototypeVariantB = () => {
  const { memory, run } = useInterviewPrototypeStore(useShallow((s) => ({ memory: s.memory, run: s.run })));
  const waiting = memory === "none" && run !== "writing";
  const message =
    run === "idle"
      ? "Answer the questions. The agent then asks about any gaps and writes the memory here."
      : "The agent writes the memory here once its follow-ups are answered.";

  return (
    <div className="@container">
      <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
        <InterviewPrototypeChecklist grouped />
        <section className="min-w-0">
          <header className="flex min-h-9 flex-wrap items-center gap-3 border-b border-border pb-3">
            <RunLine className="flex-1" trailIcons />
            <RegenerateButton />
          </header>
          {waiting && <EmptyState size="compact" icon={NotebookPen} title="No memory yet" message={message} className="mt-4" />}
          {memory === "none" && run === "writing" && (
            <LoadingDisplay label="Writing the memory from your answers" className="mt-4 rounded-2xl border border-dashed border-border" />
          )}
          {memory !== "none" && <InterviewPrototypeMemory className="mt-4" />}
        </section>
      </div>
    </div>
  );
};
