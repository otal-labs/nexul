import { useState } from "react";
import { Check } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";

import { Input } from "@/components/ui/input";
import { menuItemClass } from "@/components/MenuItem";
import { microheaderClass } from "@/components/Microheader";
import { PersonAvatar } from "@/components/PersonAvatar";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { cn } from "@/lib/utils";

interface PersonPickerProps {
  named: string[];
  onToggle: (userId: string) => void;
}

// A checkable list of the workspace's people; a person ticked here sees the project whatever roles they hold.
export const PersonPicker = ({ named, onToggle }: PersonPickerProps) => {
  const [query, setQuery] = useState("");
  const workspace = useSelectedWorkspace();
  const { data: people } = useFetchWorkspacePeople(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const needle = query.trim().toLowerCase();
  const shown = (people ?? []).filter((person) => !needle || `${person.display_name} ${person.login}`.toLowerCase().includes(needle));
  return (
    <div className="flex flex-col gap-1">
      <Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Find a person" aria-label="Find a person" className="h-8" />
      <p className={cn(microheaderClass, "px-2 pt-1.5 pb-0.5")}>People in {workspace?.name}</p>
      <div className="flex max-h-64 flex-col gap-0.5 overflow-y-auto">
        {shown.map((person) => (
          <button key={person.user_id} type="button" className={menuItemClass} aria-pressed={named.includes(person.user_id)} onClick={() => onToggle(person.user_id)}>
            <PersonAvatar login={person.login} src={person.avatar_url} className="size-5" />
            <span className="min-w-0 flex-1 truncate">{personLabel(person)}</span>
            {named.includes(person.user_id) && <Check className="size-4 text-foreground" aria-hidden />}
          </button>
        ))}
        {people && shown.length === 0 && <p className="px-2 py-3 text-sm text-muted-foreground">Nobody matches.</p>}
      </div>
      <p className="border-t border-border px-2 pt-2 pb-1 text-xs text-muted-foreground">A person added here sees the project whatever roles they hold.</p>
    </div>
  );
};
