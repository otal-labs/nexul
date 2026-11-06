import { CircleHelp, Clock, LoaderCircle } from "lucide-react";

import { RunTimer } from "@/components/board/RunTimer";
import { useBoardCardData } from "@/hooks/useBoardCardData";
import { useRunState } from "@/hooks/TrailHooks";
import { useTicketAgentTurnStart } from "@/stores/agentStreamStore";

// The card's one mark for work on its ticket: a run queued, waiting or going, an Agent turn, or work left running after it.
export const CardRunMarks = ({ ticketId }: { ticketId: string }) => {
  const { activeTrails } = useBoardCardData();
  const runState = useRunState("ticket", ticketId, activeTrails);
  const runStartedAt = activeTrails?.started[ticketId];
  const runQueued = activeTrails?.queued[ticketId] !== undefined;
  // A mention with no play behind it is a plain turn, with no trail; the card still shows the Agent at work.
  const turnStartedAt = useTicketAgentTurnStart(ticketId);
  const turning = runState === undefined && turnStartedAt !== undefined;
  const background = activeTrails?.background[ticketId] === true && runState === undefined && !turning;

  return (
    <>
      {runState === undefined && runQueued && !turning && (
        <Clock className="size-3 shrink-0 text-info" role="img" aria-label="Play queued">
          <title>Play queued</title>
        </Clock>
      )}
      {runState === "waiting" && (
        <CircleHelp className="size-3 shrink-0 text-info" role="img" aria-label="Play waiting for an answer">
          <title>Play waiting for an answer</title>
        </CircleHelp>
      )}
      {runState !== undefined && runState !== "waiting" && (
        <LoaderCircle className="size-3 shrink-0 animate-spin motion-reduce:animate-none text-warning" role="img" aria-label="Play running">
          <title>Play running</title>
        </LoaderCircle>
      )}
      {runState !== undefined && runState !== "waiting" && runStartedAt !== undefined && (
        <span className="-ml-0.5">
          <RunTimer startedAt={runStartedAt} />
        </span>
      )}
      {turning && (
        <>
          <LoaderCircle className="size-3 shrink-0 animate-spin motion-reduce:animate-none text-warning" role="img" aria-label="Agent working">
            <title>Agent working</title>
          </LoaderCircle>
          <span className="-ml-0.5">
            <RunTimer startedAt={new Date(turnStartedAt).toISOString()} />
          </span>
        </>
      )}
      {background && (
        // The one live pulse, on the in-flight hue: the turn has ended, but a command, monitor or subagent it started still runs.
        <span
          role="img"
          aria-label="Working in the background"
          title="Working in the background in T3 Code"
          className="size-1.5 shrink-0 rounded-full bg-warning-mark animate-[status-pulse_2.4s_ease-standard_infinite] motion-reduce:animate-none"
        />
      )}
    </>
  );
};
