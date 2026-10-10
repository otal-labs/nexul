import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";
import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import { TurnSegment } from "@/components/play/TurnSegment";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { SETUP_STATE_LABEL, type SetupRunRow } from "@/models/Pairing";
import type { ActivityEntry } from "@/models/Trail";
import { cn } from "@/lib/utils";
import { segmentSetupTurn } from "@/utils/TrailTranscriptUtility";

const EMPTY: ActivityEntry[] = [];

// A folded group or a short message; the scroller's default 10rem placeholder would inflate an off-screen transcript.
const SEGMENT_ITEM = "[contain-intrinsic-size:auto_3rem]";

interface SetupOutcomeProps {
  row: SetupRunRow;
}

const SetupOutcome = ({ row }: SetupOutcomeProps) => (
  <p className="flex min-w-0 items-start gap-2 text-xs">
    <SetupStateGlyph state={row.state} />
    <span className="min-w-0 break-words">
      <span className="font-medium">{SETUP_STATE_LABEL[row.state]}</span>
      <span className="text-muted-foreground"> · {row.status}</span>
    </span>
  </p>
);

interface SetupTranscriptProps {
  row: SetupRunRow;
  runningName: string | undefined;
}

// One provider's turn read like a play run's, newest at the bottom, in a well under its row; a live turn scrolls in a fixed height.
export const SetupTranscript = ({ row, runningName }: SetupTranscriptProps) => {
  const steps = useSetupActivityStore((s) => (row.turnId && s.steps[row.turnId]) || EMPTY);
  const segments = segmentSetupTurn(steps, row.state === "running");
  const waiting = runningName ? `Waiting for its turn. ${runningName} is setting up now.` : "Waiting for its turn.";
  return (
    <section aria-label={`${row.name} transcript`} className={cn("flex min-h-0 flex-col rounded-md bg-surface-2", steps.length > 0 && "h-60")}>
      <MessageScrollerProvider key={row.provider} autoScroll>
        <MessageScroller className="min-h-0 flex-1">
          <MessageScrollerViewport>
            <MessageScrollerContent role="log" aria-live="polite" aria-label={`${row.name} steps`} className="gap-1 px-3 py-3">
              {segments.map((segment, i) => (
                <MessageScrollerItem key={i} messageId={`${row.provider}-${i}`} className={SEGMENT_ITEM}>
                  <TurnSegment segment={segment} />
                </MessageScrollerItem>
              ))}
              {row.state === "running" && steps.length === 0 && <LoadingDisplay label={row.status} className="justify-start p-0" />}
              {row.state === "queued" && <EmptyRow className="text-left">{waiting}</EmptyRow>}
              {(row.state === "confirmed" || row.state === "failed") && (
                <MessageScrollerItem messageId={`${row.provider}-outcome`} className={cn(SEGMENT_ITEM, steps.length > 0 && "mt-2 border-t border-border px-1 pt-3")}>
                  <SetupOutcome row={row} />
                </MessageScrollerItem>
              )}
            </MessageScrollerContent>
          </MessageScrollerViewport>
          <MessageScrollerButton className="size-8" />
        </MessageScroller>
      </MessageScrollerProvider>
    </section>
  );
};
