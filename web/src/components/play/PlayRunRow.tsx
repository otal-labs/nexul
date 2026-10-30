import { WorkflowIcon, ZapIcon } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";

import { formatShortAgo } from "@/components/doc/docTime";
import { PersonAvatar } from "@/components/PersonAvatar";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { trailSummary, type PlayRun } from "@/models/Trail";

interface PlayRunRowProps {
  run: PlayRun;
  onOpen: (trailId: string) => void;
}

// Who started the run in a fixed left-aligned column, dropped when the runs are too narrow to keep the summary readable.
const Starter = ({ run }: { run: PlayRun }) => {
  const person = usePerson(run.starter_id);
  const auto = run.auto_play_id !== "";
  const automation = !auto && run.via === "automation";
  const name = personLabel(person);

  return (
    <span className="hidden w-32 shrink-0 items-center gap-1.5 text-xs text-muted-foreground @min-[44rem]:flex">
      {auto && <ZapIcon aria-hidden className="size-3.5 shrink-0" />}
      {auto && <span>Auto</span>}
      {automation && <WorkflowIcon aria-hidden className="size-3.5 shrink-0" />}
      {automation && <span>Automation</span>}
      {!auto && !automation && <PersonAvatar login={person.login} name={name} src={person.avatar_url} label={name} className="size-5" />}
      {!auto && !automation && <span className="min-w-0 truncate">{name}</span>}
    </span>
  );
};

// One hairline row per run; the target's key leads only for ticket runs, the only targets that have one.
export const PlayRunRow = ({ run, onOpen }: PlayRunRowProps) => {
  const { data: me } = useFetchMe();
  const starter = personLabel(usePerson(run.starter_id));
  const waiting = run.state === "waiting";
  const mine = me?.user.id === run.starter_id;
  // A waiting run names whose answer it needs, by first name, short enough to read beside a long title.
  const whose = mine ? "your" : `${starter.split(" ")[0]}'s`;
  const summary = waiting ? `Needs ${whose} answer` : trailSummary(run.state, run.last_error, run.activity[run.activity.length - 1]);
  const time = <span className="w-14 shrink-0 text-right font-mono text-xs text-muted-foreground tabular-nums">{formatShortAgo(run.started_at)}</span>;
  const key = run.target_key !== "" && <span className="w-24 shrink-0 truncate font-mono text-xs whitespace-nowrap text-muted-foreground">{run.target_key}</span>;

  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(run.id)}
        title={`${run.target_title} · ${summary}`}
        className="inset-focus flex h-10 w-full items-center gap-3 px-3 text-left transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <TrailStateIcon state={run.state} />
        {key}
        <span className="flex min-w-0 flex-1 items-baseline gap-2 text-[13px]">
          <span dir="auto" className="min-w-0 shrink truncate font-medium text-foreground">
            {run.target_title}
          </span>
          <span className="hidden max-w-[35%] shrink-0 truncate text-xs text-muted-foreground @min-[36rem]:block">{summary}</span>
        </span>
        <Starter run={run} />
        {time}
      </button>
    </li>
  );
};
