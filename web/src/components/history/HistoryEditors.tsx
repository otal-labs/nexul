import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const MAX_AVATARS = 3;

// "mcp" is an agent's save; anything else names the automation it came through.
const viaLabel = (via: string) => (via === "mcp" ? "via an agent" : `via ${via}`);

interface HistoryEditorsProps {
  editorIds: string[];
  via: string;
}

// Who a version credits, and what it came through; a sitting credits everyone who edited.
export const HistoryEditors = ({ editorIds, via }: HistoryEditorsProps) => {
  const person = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const people = editorIds.map(person);

  return (
    <span className="mt-1 flex items-start gap-1.5 text-sm">
      {people.length > 0 && (
        <span className="mt-0.5 flex shrink-0 -space-x-1">
          {people.slice(0, MAX_AVATARS).map((p) => (
            <PersonAvatar key={p.user_id} login={p.login} src={p.avatar_url} label={personLabel(p)} className="size-4 ring-1 ring-popover" />
          ))}
        </span>
      )}
      <span className="min-w-0 break-words">
        {people.map(personLabel).join(", ")}
        {via && <span className="text-muted-foreground">{people.length > 0 ? " " : ""}{viaLabel(via)}</span>}
      </span>
    </span>
  );
};
