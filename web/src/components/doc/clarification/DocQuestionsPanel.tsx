import { ClarificationActions } from "@/components/doc/clarification/ClarificationActions";
import { ClarificationRoundsFeed } from "@/components/doc/clarification/ClarificationRoundsFeed";
import { ClarificationStatusLine } from "@/components/doc/clarification/ClarificationStatusLine";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDoc, useFetchDocClarification } from "@/hooks/DocHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { clarificationPhase } from "@/models/Clarification";

interface DocQuestionsPanelProps {
  docId: string;
}

// The doc page's Questions view in the article's place: the heading with the state line and, for people who may close
// it, the next action, then every round. A locked doc still takes answers, which are not edits to it.
export const DocQuestionsPanel = ({ docId }: DocQuestionsPanelProps) => {
  const { data: clarification, error, isPending } = useFetchDocClarification(docId);
  const { data: doc } = useFetchDoc(docId);
  const canWrite = useHasPermission("docs:write");
  return (
    <div className="rounded-lg border border-border bg-card p-6 shadow-card sm:p-8">
      <section aria-label="Questions" className="mx-auto min-w-0 max-w-3xl">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {clarification && (
          <header className="flex flex-wrap items-end justify-between gap-3 border-b border-border pb-3">
            <div className="min-w-0 flex-1 basis-56 space-y-1">
              <h2 className="text-base font-semibold tracking-tight">Questions</h2>
              <ClarificationStatusLine clarification={clarification} locked={doc?.locked ?? false} />
            </div>
            {clarification.can_close && <ClarificationActions clarification={clarification} docId={docId} />}
          </header>
        )}
        {clarification && clarification.can_close && clarificationPhase(clarification) === "none" && (
          <EmptyRow className="mt-5">Clarify via AI asks the doc's authors about what it leaves open, a round at a time.</EmptyRow>
        )}
        {clarification && <ClarificationRoundsFeed clarification={clarification} readOnly={!canWrite} />}
      </section>
    </div>
  );
};
