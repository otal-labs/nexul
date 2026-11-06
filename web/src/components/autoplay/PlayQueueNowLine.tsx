import { personLabel } from "@nexul/client-core/person";

import { useFetchPlayQueued } from "@/hooks/PlayQueueHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { Play } from "@/models/Play";
import { waitingGroups } from "@/utils/PlayQueueUtility";

// Past this many people the line says how many more wait instead of naming each.
const NAMED = 2;

// The queue's plain "next in line" says nothing is wrong; here it reads as what the runs are doing.
const whyText = (why: string) => (why === "next in line" ? "waiting their turn" : why);

// What this play has queued now, by whose computers each run waits for and why: "2 queued for Alice's computers, waiting their turn".
export const PlayQueueNowLine = ({ play }: { play: Play }) => {
  const { data: queued } = useFetchPlayQueued(play.id);
  const person = usePersonLookup(play.workspace_id);
  const groups = waitingGroups(queued ?? []);
  const rest = groups.slice(NAMED).reduce((n, g) => n + g.count, 0);
  const one = groups.length === 1 ? groups[0] : undefined;

  return (
    queued &&
    queued.length > 0 && (
      <p className="text-xs text-muted-foreground [overflow-wrap:anywhere]">
        <span className="font-mono tabular-nums">{queued.length}</span> queued
        {one && (
          <>
            {" for "}
            <span className="text-foreground">{personLabel(person(one.personId))}'s</span> computers, {whyText(one.why)}
          </>
        )}
        {!one && ":"}
        {!one &&
          groups.slice(0, NAMED).map((g, i) => (
            <span key={`${g.personId} ${g.why}`}>
              {i > 0 && " ·"} <span className="font-mono tabular-nums">{g.count}</span> for{" "}
              <span className="text-foreground">{personLabel(person(g.personId))}'s</span> computers ({whyText(g.why)})
            </span>
          ))}
        {rest > 0 && (
          <>
            {" · "}
            <span className="font-mono tabular-nums">{rest}</span> more
          </>
        )}
      </p>
    )
  );
};
