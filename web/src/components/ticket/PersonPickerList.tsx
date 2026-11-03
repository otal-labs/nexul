import { CheckIcon } from "lucide-react";
import { useState } from "react";

import { personLabel } from "@nexul/client-core/person";

import { NoPersonAvatar } from "@/components/NoPersonAvatar";
import { PersonAvatar } from "@/components/PersonAvatar";
import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Input } from "@/components/ui/input";
import { useFetchProjectPeople } from "@/hooks/PeopleHooks";

interface PersonPickerListProps {
  // The ticket's project: only people who may open it can develop or test it.
  projectId: string | undefined;
  onSelect: (login: string) => void;
  // The login already in the role, checked in the list; empty checks No one.
  selected?: string;
}

const selectedMark = <CheckIcon className="ml-auto size-3.5 shrink-0" aria-hidden />;

// Shared member list body for every popover; callers own the Popover/trigger around it.
export const PersonPickerList = ({ projectId, onSelect, selected }: PersonPickerListProps) => {
  const { data: people } = useFetchProjectPeople(projectId);
  const [search, setSearch] = useState("");
  const query = search.trim().toLowerCase();
  const filtered = people?.filter((p) => `${p.login} ${p.display_name}`.toLowerCase().includes(query));

  return (
    <>
      <Input
        autoFocus
        aria-label="Search people"
        placeholder="Search people…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="mb-1.5 h-8 text-xs"
      />
      <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">
        {/* Hidden while searching, so a query's matches never sit under a row that clears the role. */}
        {query === "" && (
          <button type="button" aria-pressed={selected === ""} className={menuItemClass} onClick={() => onSelect("")}>
            <NoPersonAvatar />
            No one
            {selected === "" && selectedMark}
          </button>
        )}
        {filtered?.map((member) => (
          <button
            key={member.user_id}
            type="button"
            aria-pressed={selected === member.login}
            className={menuItemClass}
            onClick={() => onSelect(member.login)}
          >
            <PersonAvatar login={member.login} name={personLabel(member)} src={member.avatar_url} />
            <span title={personLabel(member)} className="min-w-0 truncate">{personLabel(member)}</span>
            {selected === member.login && selectedMark}
          </button>
        ))}
        {filtered && filtered.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>}
      </div>
    </>
  );
};
