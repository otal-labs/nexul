import { SettingsIcon } from "lucide-react";

import { DocTicketsSection } from "@/components/doc/DocTicketsSection";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSetDocLocked } from "@/hooks/DocHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { Doc } from "@/models/Doc";

interface DocActionsMenuProps {
  doc: Doc;
  onCreateTicket: () => void;
  onPermissions: () => void;
  onArchive: () => void;
  onRestore: () => void;
}

export const DocActionsMenu = ({
  doc,
  onCreateTicket,
  onPermissions,
  onArchive,
  onRestore,
}: DocActionsMenuProps) => {
  const canLock = useHasPermission("docs:lock");
  const setLocked = useSetDocLocked();
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="size-7" aria-label="Doc actions">
          <SettingsIcon className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-56 p-1">
        <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onCreateTicket}>
          Create ticket from this doc
        </Button>
        <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onPermissions}>
          Permissions
        </Button>
        {canLock && !doc.locked && (
          <Button variant="ghost" size="sm" className="w-full justify-start" onClick={() => setLocked.mutate({ id: doc.id, locked: true })}>
            Lock
          </Button>
        )}
        {canLock && doc.locked && (
          <Button variant="ghost" size="sm" className="w-full justify-start" onClick={() => setLocked.mutate({ id: doc.id, locked: false })}>
            Unlock
          </Button>
        )}
        {!doc.archived && (
          <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onArchive}>
            Archive
          </Button>
        )}
        {doc.archived && (
          <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onRestore}>
            Restore
          </Button>
        )}
        <div className="mt-2 border-t border-border px-2 pt-2">
          <DocTicketsSection docId={doc.id} />
        </div>
      </PopoverContent>
    </Popover>
  );
};
