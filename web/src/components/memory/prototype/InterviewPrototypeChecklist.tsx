import { useShallow } from "zustand/react/shallow";

import { FOLLOW_UPS, TEMPLATE, type ProtoQuestion } from "@/components/memory/prototype/InterviewPrototypeData";
import { InterviewPrototypeRow } from "@/components/memory/prototype/InterviewPrototypeRow";
import { InterviewPrototypeSection } from "@/components/memory/prototype/InterviewPrototypeSection";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { cn } from "@/lib/utils";

interface ChecklistProps {
  className?: string;
  // grouped puts the template and the follow-ups in collapsible sections (variant B).
  grouped?: boolean;
}

const TemplateRows = ({ className }: { className?: string }) => (
  <ol className={className}>
    {TEMPLATE.map((q, i) => (
      <InterviewPrototypeRow key={q.id} question={q} number={i + 1} first={i === 0} progress={`Question ${i + 1} of ${TEMPLATE.length}`} />
    ))}
  </ol>
);

const FollowUpRows = ({ followUps, className }: { followUps: ProtoQuestion[]; className?: string }) => (
  <ol className={className}>
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
);

const countLine = (questions: ProtoQuestion[], answers: Record<string, unknown>, skipped: string[]): string => {
  const answered = questions.filter((q) => answers[q.id] !== undefined).length;
  const skips = questions.filter((q) => skipped.includes(q.id)).length;
  if (skips === 0) return `${answered} of ${questions.length} answered`;
  return `${answered} answered · ${skips} skipped`;
};

// The template's questions as numbered rows, then the agent's follow-ups under their own label.
export const InterviewPrototypeChecklist = ({ className, grouped = false }: ChecklistProps) => {
  const { followUpCount, answers, skipped, legacy } = useInterviewPrototypeStore(
    useShallow((s) => ({
      followUpCount: s.followUps,
      answers: s.answers,
      skipped: s.skipped,
      legacy: s.memory === "legacy" && Object.keys(s.answers).length === 0,
    })),
  );
  const followUps = FOLLOW_UPS.slice(0, followUpCount);

  return (
    <div className={cn("min-w-0 max-w-[40rem]", className)}>
      {legacy && (
        <p className="mb-3 text-sm text-muted-foreground">
          This memory came from an earlier interview. Answering these lets the agent update it.
        </p>
      )}
      {!grouped && <TemplateRows className="border-t border-border" />}
      {!grouped && followUps.length > 0 && (
        <>
          <p className="mt-8 mb-2 font-mono text-[11px] tracking-wide text-muted-foreground uppercase">Follow-ups from the agent</p>
          <FollowUpRows followUps={followUps} className="border-t border-border" />
        </>
      )}
      {grouped && (
        <InterviewPrototypeSection sectionKey="initial" label="Initial questions" meta={countLine(TEMPLATE, answers, skipped)}>
          <TemplateRows />
        </InterviewPrototypeSection>
      )}
      {grouped && followUps.length > 0 && (
        <InterviewPrototypeSection
          sectionKey="followups"
          label="Follow-ups from the agent"
          meta={countLine(followUps, answers, skipped)}
          className="mt-6"
        >
          <FollowUpRows followUps={followUps} />
        </InterviewPrototypeSection>
      )}
    </div>
  );
};
