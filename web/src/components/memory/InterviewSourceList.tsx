import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { InterviewSourceRow } from "@/components/memory/InterviewSourceRow";
import { useFetchInterviewDrafts, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { SOURCE_STANCE_LABEL, citationCounts, lastDraftedAt, sourceChange, type InterviewSource, type SourceStance } from "@/models/InterviewSource";

interface InterviewSourceListProps {
  projectId: string;
  sources: InterviewSource[];
  readOnly: boolean;
}

interface StanceGroupProps extends InterviewSourceListProps {
  stance: SourceStance;
}

const StanceGroup = ({ projectId, sources, readOnly, stance }: StanceGroupProps) => {
  const drafting = useInterviewTrails(projectId).drafting;
  const { data: drafts } = useFetchInterviewDrafts(projectId);
  const since = drafting && lastDraftedAt(drafting);
  const cites = citationCounts(drafts ?? []);
  const rows = sources.filter((s) => s.stance === stance);
  if (rows.length === 0) return null;
  return (
    <section aria-label={SOURCE_STANCE_LABEL[stance]} className="space-y-1">
      <Microheader className="flex items-baseline gap-2 px-1.5">
        {SOURCE_STANCE_LABEL[stance]}
        <span className="tabular-nums">{rows.length}</span>
      </Microheader>
      <ul>
        {rows.map((source) => (
          <InterviewSourceRow key={source.id} source={source} change={sourceChange(source, since)} cites={cites.get(source.id) ?? 0} readOnly={readOnly} />
        ))}
      </ul>
    </section>
  );
};

const emptyClass = "border-0 px-1.5 py-1 text-left";

// The sources in two groups by stance: what the agent drafts from, then what the follow-ups ask about.
export const InterviewSourceList = ({ projectId, sources, readOnly }: InterviewSourceListProps) => (
  <div className="space-y-4">
    {sources.length === 0 && readOnly && <EmptyRow className={emptyClass}>No sources.</EmptyRow>}
    {sources.length === 0 && !readOnly && (
      <EmptyRow className={emptyClass}>No sources yet. Add a practices folder, a standards doc, or the project this one replaces.</EmptyRow>
    )}
    <StanceGroup projectId={projectId} sources={sources} readOnly={readOnly} stance="follow" />
    <StanceGroup projectId={projectId} sources={sources} readOnly={readOnly} stance="question" />
  </div>
);
