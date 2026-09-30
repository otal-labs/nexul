import { LockIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useSetDocLocked } from "@/hooks/DocHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

interface DocLockedSignalProps {
  docId: string;
}

// Says the doc is read-only and offers the way back to whoever may edit it.
export const DocLockedSignal = ({ docId }: DocLockedSignalProps) => {
  const canWrite = useHasPermission("docs:write");
  const setLocked = useSetDocLocked();
  return (
    <span className="flex items-center gap-1">
      <span className="flex items-center gap-1.5 px-1 font-mono text-xs text-muted-foreground">
        <LockIcon className="size-3.5" aria-hidden />
        Locked
      </span>
      {canWrite && (
        <Button
          variant="ghost"
          size="sm"
          loading={setLocked.isPending}
          onClick={() => setLocked.mutate({ id: docId, locked: false })}
        >
          Unlock
        </Button>
      )}
    </span>
  );
};
