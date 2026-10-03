import { FOLLOW_UPS, TEMPLATE } from "@/components/memory/prototype/InterviewPrototypeData";
import { InterviewPrototypeRow } from "@/components/memory/prototype/InterviewPrototypeRow";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { cn } from "@/lib/utils";

// The template's questions as numbered rows, then the agent's follow-ups under their own label.
export const InterviewPrototypeChecklist = ({ className }: { className?: string }) => {
  const followUps = FOLLOW_UPS.slice(0, useInterviewPrototypeStore((s) => s.followUps));
  const legacy = useInterviewPrototypeStore((s) => s.memory === "legacy" && Object.keys(s.answers).length === 0);

  return (
    <div className={cn("min-w-0 max-w-[40rem]", className)}>
      {legacy && (
        <p className="mb-3 text-sm text-muted-foreground">
          This memory came from an earlier interview. Answering these lets the agent update it.
        </p>
      )}
      <ol className="border-t border-border">
        {TEMPLATE.map((q, i) => (
          <InterviewPrototypeRow
            key={q.id}
            question={q}
            number={i + 1}
            first={i === 0}
            progress={`Question ${i + 1} of ${TEMPLATE.length}`}
          />
        ))}
      </ol>
      {followUps.length > 0 && (
        <>
          <p className="mt-8 mb-2 font-mono text-[11px] tracking-wide text-muted-foreground uppercase">Follow-ups from the agent</p>
          <ol className="border-t border-border">
            {followUps.map((q, i) => (
              <InterviewPrototypeRow
                key={q.id}
                question={q}
                number={TEMPLATE.length + i + 1}
                first={false}
                progress={`Follow-up ${i + 1} of ${followUps.length}`}
              />
            ))}
          </ol>
        </>
      )}
    </div>
  );
};
