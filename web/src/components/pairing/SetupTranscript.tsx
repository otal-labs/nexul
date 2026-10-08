import { RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
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
  retryDisabled: boolean;
  onRetry: (provider: string) => void;
}

const SetupOutcome = ({ row, retryDisabled, onRetry }: SetupOutcomeProps) => (
  <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
    <p className="flex min-w-0 items-start gap-2 text-xs">
      <SetupStateGlyph state={row.state} />
      <span className="min-w-0 break-words">
        <span className="font-medium">{SETUP_STATE_LABEL[row.state]}</span>
        <span className="text-muted-foreground"> · {row.status}</span>
      </span>
    </p>
    {row.state === "failed" && (
      <Button type="button" variant="outline" size="sm" disabled={retryDisabled} onClick={() => onRetry(row.provider)}>
        <RotateCcw className="size-3.5" aria-hidden />
        Retry {row.name}
      </Button>
    )}
  </div>
);

interface SetupTranscriptProps {
  row: SetupRunRow | undefined;
  runningName: string | undefined;
  retryDisabled: boolean;
  onRetry: (provider: string) => void;
}

// The picked provider's live turn read like a play run's, newest at the bottom; the scroller follows the end until the reader scrolls up.
export const SetupTranscript = ({ row, runningName, retryDisabled, onRetry }: SetupTranscriptProps) => {
  const steps = useSetupActivityStore((s) => (row?.turnId && s.steps[row.turnId]) || EMPTY);
  const segments = segmentSetupTurn(steps, row?.state === "running");
  const waiting = runningName ? `Waiting for its turn. ${runningName} is setting up now.` : "Waiting for its turn.";
  return (
    <section
      aria-label="Setup transcript"
      className="flex min-h-0 min-w-0 flex-col border-border max-md:mt-5 max-md:h-80 max-md:rounded-lg max-md:border md:flex-1 md:border-l"
    >
      {!row && (
        <EmptyRow className="m-4 text-left">
          Nothing has run here yet. Start setup and each provider takes one turn: it connects Nexul's MCP server with this
          computer's own token, installs the skills, and confirms the skills it found. Its steps stream here as it works.
        </EmptyRow>
      )}
      {row && (
        <header className="flex items-center gap-2 border-b border-border px-4 py-2.5">
          <SetupStateGlyph state={row.state} />
          <h3 className="min-w-0 truncate text-sm font-medium">{row.name}</h3>
          <span className="text-xs text-muted-foreground">
            {SETUP_STATE_LABEL[row.state]}
            {row.kind === "skills" && " · skills update"}
          </span>
          {row.model && <span className="ml-auto truncate font-mono text-[11px] text-muted-foreground">{row.model}</span>}
        </header>
      )}
      {row && (
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
                    <SetupOutcome row={row} retryDisabled={retryDisabled} onRetry={onRetry} />
                  </MessageScrollerItem>
                )}
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <MessageScrollerButton className="size-8" />
          </MessageScroller>
        </MessageScrollerProvider>
      )}
    </section>
  );
};
