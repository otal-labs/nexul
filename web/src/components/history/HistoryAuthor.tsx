import { BotIcon, WorkflowIcon } from "lucide-react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { useMinWidth } from "@/hooks/useMinWidth";
import { viaLabel } from "@/utils/HistoryVersionUtility";

interface HistoryAuthorProps {
  /** Everyone the version credits, the author first. */
  people: Person[];
  via: string;
}

// What a version came through is drawn the way the rest of the app draws it: the bot for an agent ("mcp", as in chat), the
// workflow mark for an automation or a play.
const ViaMark = ({ via, className }: { via: string; className: string }) => {
  const Mark = via === "mcp" ? BotIcon : WorkflowIcon;
  return <Mark aria-hidden className={className} />;
};

// The first editor on one line, up to two small stacked avatars for the rest (all of them in the tooltip and the row's accessible
// name), then what it came through: the mark and its
// words from 1024px up, the mark alone below, so the name keeps its width (the phrase stays in the tooltip and the row's
// accessible name). An automation has no editor, so a mark in a well holds the avatar's place and the names line up.
export const HistoryAuthor = ({ people, via }: HistoryAuthorProps) => {
  const wide = useMinWidth(1024);
  const [first] = people;
  return (
    <span className="mt-1.5 flex items-center gap-1.5" title={[people.map(personLabel).join(", "), via && viaLabel(via)].filter(Boolean).join(" ") || undefined}>
      {first && <PersonAvatar login={first.login} src={first.avatar_url} label={personLabel(first)} className="size-[18px]" />}
      {!first && (
        <span aria-hidden className="flex size-[18px] shrink-0 items-center justify-center rounded-full bg-accent text-muted-foreground">
          <ViaMark via={via} className="size-3" />
        </span>
      )}
      <span className="flex min-w-0 items-center gap-1.5 text-[13px] leading-[18px]">
        {first && <span className="min-w-0 truncate">{personLabel(first)}</span>}
        {people.length > 1 && (
          <span aria-hidden className="flex shrink-0 -space-x-1">
            {people.slice(1, 3).map((p) => (
              <PersonAvatar key={p.user_id} login={p.login} src={p.avatar_url} label={personLabel(p)} className="size-3.5 ring-1 ring-popover" />
            ))}
          </span>
        )}
        {via && first && (
          <span className="flex shrink-0 items-center gap-1 text-muted-foreground">
            <ViaMark via={via} className="size-4 shrink-0" />
            {wide && viaLabel(via)}
          </span>
        )}
        {via && !first && <span className="min-w-0 truncate text-muted-foreground">{viaLabel(via)}</span>}
      </span>
    </span>
  );
};
