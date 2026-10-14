import { LockIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useSetDocLocked } from "@/hooks/DocHooks";
import { useEntityActions } from "@/hooks/useEntityActions";

interface DocLockedSignalProps {
  docId: string;
}

// Says the doc is read-only and offers the way back to whoever may unlock it.
export const DocLockedSignal = ({ docId }: DocLockedSignalProps) => {
  const answer = useEntityActions("doc", docId);
  const setLocked = useSetDocLocked();
  return (
    <span className="flex items-center gap-1">
      <span className="flex items-center gap-1.5 px-1 font-mono text-xs text-muted-foreground">
        <LockIcon className="size-3.5" aria-hidden />
        Locked
      </span>
      <ActionButton
        permission="docs:lock"
        answer={answer("docs:lock")}
        variant="ghost"
        size="sm"
        loading={setLocked.isPending}
        onClick={() => setLocked.mutate({ id: docId, locked: false })}
      >
        Unlock
      </ActionButton>
    </span>
  );
};
