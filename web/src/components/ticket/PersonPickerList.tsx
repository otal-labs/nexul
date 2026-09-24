import { useState } from "react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Input } from "@/components/ui/input";
import { useFetchWorkspaceMembers } from "@/hooks/MemberHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const menuItemClass =
  "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs text-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:bg-accent/60";

interface PersonPickerListProps {
  onSelect: (login: string) => void;
}

// Shared member list body for every popover; callers own the Popover/trigger around it.
export const PersonPickerList = ({ onSelect }: PersonPickerListProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: membersList } = useFetchWorkspaceMembers(workspaceId);
  const [search, setSearch] = useState("");
  const members = membersList?.members ?? [];
  const filtered = members.filter((m) => m.login.toLowerCase().includes(search.trim().toLowerCase()));

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
        <button type="button" className={menuItemClass} onClick={() => onSelect("")}>
          No one
        </button>
        {filtered.map((member) => (
          <button
            key={member.user_id}
            type="button"
            className={menuItemClass}
            onClick={() => onSelect(member.login)}
          >
            <PersonAvatar login={member.login} />
            {member.login}
          </button>
        ))}
        {filtered.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>}
      </div>
    </>
  );
};
