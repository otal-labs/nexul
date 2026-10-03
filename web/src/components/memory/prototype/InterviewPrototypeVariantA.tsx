import { InterviewPrototypeChecklist } from "@/components/memory/prototype/InterviewPrototypeChecklist";
import { InterviewPrototypeMemory } from "@/components/memory/prototype/InterviewPrototypeMemory";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { InterviewPrototypeTimeline } from "@/components/memory/prototype/InterviewPrototypeTimeline";

// A: the checklist with a narrow side card for the run; the memory under both, full width, once it exists.
export const InterviewPrototypeVariantA = () => {
  const hasMemory = useInterviewPrototypeStore((s) => s.memory !== "none");
  return (
    <div className="@container">
      <div className="grid gap-8 @2xl:grid-cols-[minmax(0,40rem)_14rem] @2xl:items-start @2xl:gap-10">
        <InterviewPrototypeChecklist className="order-2 @2xl:order-1" />
        <InterviewPrototypeTimeline className="order-1 @2xl:order-2" />
      </div>
      {hasMemory && (
        <section className="mt-14">
          <p className="mb-3 font-mono text-[11px] tracking-wide text-muted-foreground uppercase">Interview memory</p>
          <InterviewPrototypeMemory />
        </section>
      )}
    </div>
  );
};
