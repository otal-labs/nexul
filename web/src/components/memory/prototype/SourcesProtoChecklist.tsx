import { InterviewSection } from "@/components/memory/InterviewSection";
import { SourcesProtoQuestion } from "@/components/memory/prototype/SourcesProtoQuestion";
import { SourcesProtoRow } from "@/components/memory/prototype/SourcesProtoRow";
import { SourcesProtoSuggestion } from "@/components/memory/prototype/SourcesProtoSuggestion";
import { FOLLOW_UPS, QUESTIONS } from "@/components/memory/prototype/SourcesProtoData";
import { rowsOf, rowState, useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import type { QuestionItem } from "@/models/Question";

interface Section {
  key: string;
  label: string;
  items: QuestionItem[];
}

const SECTIONS: Section[] = [
  { key: "initial", label: "Initial questions", items: QUESTIONS },
  { key: "f1", label: "Follow-ups from the agent 1", items: FOLLOW_UPS },
];

const ProtoItem = ({ item, progress }: { item: QuestionItem; progress: string }) => {
  const followUps = useSourcesProtoStore((s) => s.followUps);
  const suggestion = useSourcesProtoStore((s) => s.suggestions[item.id]);
  const rows = rowsOf(followUps);
  const at = rows.indexOf(item);
  return (
    <SourcesProtoRow item={item} number={at + 1}>
      {suggestion && <SourcesProtoSuggestion id={item.id} progress={progress} suggestion={suggestion} />}
      {!suggestion && <SourcesProtoQuestion item={item} progress={progress} prevId={rows[at - 1]?.id ?? null} />}
    </SourcesProtoRow>
  );
};

const ProtoSection = ({ section, newest }: { section: Section; newest: boolean }) => {
  const s = useSourcesProtoStore();
  const states = section.items.map((i) => rowState(s, i.id));
  const answered = states.filter((x) => x === "answered").length;
  const drafted = states.filter((x) => x === "drafted").length;
  const suggested = section.items.filter((i) => s.suggestions[i.id]).length;
  const meta = [`${answered} of ${section.items.length} answered`, drafted > 0 && `${drafted} drafted`, suggested > 0 && `${suggested} suggested`]
    .filter(Boolean)
    .join(" · ");
  const folded = s.folds[section.key] ?? (!newest && !section.items.some((i) => i.id === s.openId));
  return (
    <InterviewSection label={section.label} meta={meta} folded={folded} onToggle={() => s.toggleFold(section.key, !folded)}>
      <ol>
        {section.items.map((item, i) => (
          <ProtoItem key={item.id} item={item} progress={`${section.key === "initial" ? "Question" : "Follow-up"} ${i + 1} of ${section.items.length}`} />
        ))}
      </ol>
    </InterviewSection>
  );
};

// The question checklist with drafts and suggested changes mocked in.
export const SourcesProtoChecklist = () => {
  const followUps = useSourcesProtoStore((s) => s.followUps);
  const sections = followUps > 0 ? SECTIONS : SECTIONS.slice(0, 1);
  return (
    <div className="min-w-0 space-y-6">
      {sections.map((section, i) => (
        <ProtoSection key={section.key} section={section} newest={i === sections.length - 1} />
      ))}
    </div>
  );
};
