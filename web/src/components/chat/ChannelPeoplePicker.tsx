import { SearchIcon } from "lucide-react";
import { useState } from "react";

import { Input } from "@/components/ui/input";
import { ChannelPersonOption } from "@/components/chat/ChannelPersonOption";
import { EmptyRow } from "@/components/EmptyRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { personLabel } from "@/models/Person";

interface ChannelPeoplePickerProps {
  value: string[];
  onChange: (userIds: string[]) => void;
  // People already in the channel, left out of the list.
  excludeIds?: string[];
  // Whoever creates or switches a channel stays in it, so their row is checked and fixed.
  keepsViewer?: boolean;
}

export const ChannelPeoplePicker = ({ value, onChange, excludeIds = [], keepsViewer = true }: ChannelPeoplePickerProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: me } = useFetchMe();
  const { data: people } = useFetchWorkspacePeople(workspaceId);
  const [search, setSearch] = useState("");
  const meId = keepsViewer ? me?.user.id : undefined;
  const shown = (people ?? []).filter((p) => !excludeIds.includes(p.user_id));
  const sorted = [...shown.filter((p) => p.user_id === meId), ...shown.filter((p) => p.user_id !== meId)];
  const query = search.trim().toLowerCase();
  const matches = sorted.filter((p) => `${p.login} ${personLabel(p)}`.toLowerCase().includes(query));
  const picked = shown.filter((p) => p.user_id === meId || value.includes(p.user_id)).length;
  const toggle = (userId: string, on: boolean) => onChange(on ? [...value, userId] : value.filter((id) => id !== userId));

  return (
    <div>
      <div className="flex min-h-11 items-center gap-3 py-1.5">
        <p className="min-w-0 flex-1 truncate font-mono text-[11px] tracking-wide text-muted-foreground uppercase">
          {picked} of {shown.length} people
        </p>
        <div className="relative w-44 shrink-0">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            aria-label="Search people"
            placeholder="Search people"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="h-8 pl-8 text-[13px]"
          />
        </div>
      </div>
      {matches.length === 0 && <EmptyRow className="py-4">{shown.length === 0 ? "Everyone is already in it." : "Nobody matches."}</EmptyRow>}
      {matches.length > 0 && (
        <ul className="max-h-64 divide-y divide-border overflow-y-auto border-y border-border">
          {matches.map((person) => (
            <ChannelPersonOption
              key={person.user_id}
              person={person}
              isYou={person.user_id === meId}
              checked={person.user_id === meId || value.includes(person.user_id)}
              onCheckedChange={(on) => toggle(person.user_id, on)}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
