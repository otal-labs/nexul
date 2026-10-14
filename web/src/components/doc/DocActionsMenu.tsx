import { SettingsIcon } from "lucide-react";

import { DocTicketsSection } from "@/components/doc/DocTicketsSection";
import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSetDocLocked } from "@/hooks/DocHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { Doc } from "@/models/Doc";

interface DocActionsMenuProps {
  doc: Doc;
  onCreateTicket: () => void;
  onPermissions: () => void;
  onArchive: () => void;
  onRestore: () => void;
}

export const DocActionsMenu = ({ doc, onCreateTicket, onPermissions, onArchive, onRestore }: DocActionsMenuProps) => {
  const answer = useEntityActions("doc", doc.id);
  const setLocked = useSetDocLocked();
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="size-7" aria-label="Doc actions">
          <SettingsIcon className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-56 p-1">
        <ActionMenuItem permission="tickets:write" projectId={doc.project_id} verb="create tickets" onSelect={onCreateTicket}>
          Create ticket from this doc
        </ActionMenuItem>
        <ActionMenuItem permission="permissions:write" answer={answer("permissions:write")} verb="manage this doc's permissions" onSelect={onPermissions}>
          Permissions…
        </ActionMenuItem>
        {!doc.locked && (
          <ActionMenuItem permission="docs:lock" answer={answer("docs:lock")} verb="lock docs" onSelect={() => setLocked.mutate({ id: doc.id, locked: true })}>
            Lock
          </ActionMenuItem>
        )}
        {doc.locked && (
          <ActionMenuItem permission="docs:lock" answer={answer("docs:lock")} verb="unlock docs" onSelect={() => setLocked.mutate({ id: doc.id, locked: false })}>
            Unlock
          </ActionMenuItem>
        )}
        {!doc.archived && (
          <ActionMenuItem permission="docs:write" answer={answer("docs:write")} verb="archive docs" onSelect={onArchive}>
            Archive
          </ActionMenuItem>
        )}
        {doc.archived && (
          <ActionMenuItem permission="docs:write" answer={answer("docs:write")} verb="restore docs" onSelect={onRestore}>
            Restore
          </ActionMenuItem>
        )}
        <div className="mt-2 border-t border-border px-2 pt-2">
          <DocTicketsSection docId={doc.id} />
        </div>
      </PopoverContent>
    </Popover>
  );
};
