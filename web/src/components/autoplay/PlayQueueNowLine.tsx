import { personLabel } from "@nexul/client-core/person";

import { useFetchPlayQueued } from "@/hooks/PlayQueueHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { Play } from "@/models/Play";
import { waitingGroups } from "@/utils/PlayQueueUtility";

// Past this many people the line says how many more wait instead of naming each.
const NAMED = 2;

export const PlayQueueNowLine = ({ play }: { play: Play }) => {
  const { data: queued } = useFetchPlayQueued(play.id);
  const person = usePersonLookup(play.workspace_id);
  const groups = waitingGroups(queued ?? []);
  const rest = groups.slice(NAMED).reduce((n, g) => n + g.count, 0);

  return (
    queued &&
    queued.length > 0 && (
      <p className="text-xs text-muted-foreground [overflow-wrap:anywhere]">
        Right now: <span className="font-mono tabular-nums">{queued.length}</span> queued
        {groups.slice(0, NAMED).map((g) => (
          <span key={`${g.personId} ${g.why}`}>
            {" · "}
            <span className="font-mono tabular-nums">{g.count}</span> waiting on{" "}
            <span className="text-foreground">{personLabel(person(g.personId))}</span> ({g.why})
          </span>
        ))}
        {rest > 0 && (
          <>
            {" · "}
            <span className="font-mono tabular-nums">{rest}</span> more waiting
          </>
        )}
      </p>
    )
  );
};
