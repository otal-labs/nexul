import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";

import { TrailFacts } from "@/components/play/TrailFacts";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { TrailTranscript } from "@/components/play/TrailTranscript";
import { useLiveTrailActivity, useLiveTrailQuestion, useLiveTrailState, useLiveTrailSteps } from "@/hooks/TrailHooks";
import { trailSummary, type Trail } from "@/models/Trail";

interface TrailDetailBodyProps {
  trail: Trail;
}

const microheaderClass = "font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Run facts above a hairline, then the transcript filling the rest of the dialog; the scroller follows the live
// edge while the run grows and offers "Scroll to end" once the reader has scrolled away.
export const TrailDetailBody = ({ trail }: TrailDetailBodyProps) => {
  const state = useLiveTrailState(trail);
  const liveStep = useLiveTrailActivity(trail);
  const question = useLiveTrailQuestion(trail);
  const steps = useLiveTrailSteps(trail);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="space-y-4 px-6 pb-4">
        <p className="flex items-center gap-2 text-sm">
          <TrailStateIcon state={state} />
          <span className="truncate">{trailSummary(state, trail.last_error, liveStep)}</span>
        </p>
        <TrailFacts trail={trail} />
        {trail.last_error !== "" && (
          <section className="space-y-1">
            <h3 className={microheaderClass}>Last error</h3>
            <p className="text-sm text-destructive">{trail.last_error}</p>
          </section>
        )}
      </div>

      <section className="flex min-h-0 flex-1 flex-col border-t border-border">
        <h3 className={`${microheaderClass} px-6 pt-4 pb-2`}>Transcript</h3>
        <MessageScrollerProvider autoScroll>
          <MessageScroller className="min-h-0 flex-1">
            <MessageScrollerViewport className="px-4 pb-4">
              <MessageScrollerContent className="gap-0">
                <TrailTranscript trail={trail} steps={steps} state={state} question={question} />
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <MessageScrollerButton />
          </MessageScroller>
        </MessageScrollerProvider>
      </section>
    </div>
  );
};
