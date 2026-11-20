import {
  MessageScroller,
  MessageScrollerContent,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";

import { microheaderClass } from "@/components/Microheader";
import { SetupRefusalLink } from "@/components/pairing/SetupRefusalLink";
import { TrailContinueForm } from "@/components/play/TrailContinueForm";
import { TrailFacts } from "@/components/play/TrailFacts";
import { TrailTranscript } from "@/components/play/TrailTranscript";
import { TranscriptScrollButton } from "@/components/TranscriptScrollButton";
import { useLiveTrailQuestion, useLiveTrailState, useLiveTrailSteps } from "@/hooks/TrailHooks";
import { SETUP_REQUIRED_REASON } from "@/models/Pairing";
import { isTrailActive, type Trail } from "@/models/Trail";

interface TrailDetailBodyProps {
  trail: Trail;
}

// The run's facts above a hairline, then the transcript filling the rest of the dialog; the scroller follows the live
// edge while the run grows and offers "Scroll to end" once the reader has scrolled away. An ended run can be continued.
export const TrailDetailBody = ({ trail }: TrailDetailBodyProps) => {
  const state = useLiveTrailState(trail);
  const question = useLiveTrailQuestion(trail);
  const steps = useLiveTrailSteps(trail);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="space-y-5 px-6 pb-5">
        <TrailFacts trail={trail} />
        {trail.last_error !== "" && (
          <section className="space-y-1">
            <h3 className={microheaderClass}>Last error</h3>
            <p className="text-sm text-destructive">{trail.last_error}</p>
            {trail.failure_reason === SETUP_REQUIRED_REASON && <SetupRefusalLink computerId={trail.computer_id} />}
          </section>
        )}
      </div>

      <section aria-label="Transcript" className="flex min-h-0 flex-1 flex-col border-t border-border">
        <MessageScrollerProvider autoScroll>
          <MessageScroller className="min-h-0 flex-1">
            <MessageScrollerViewport className="px-4 pt-4 pb-4">
              <MessageScrollerContent className="gap-0">
                <TrailTranscript trail={trail} steps={steps} state={state} question={question} />
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <TranscriptScrollButton />
          </MessageScroller>
        </MessageScrollerProvider>
      </section>
      {!isTrailActive(state) && <TrailContinueForm trail={trail} />}
    </div>
  );
};
