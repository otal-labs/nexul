import { LockIcon } from "lucide-react";

// Says the doc is read-only; Unlock is in the doc's menu, so the header holds one quiet mark instead of a word and a button.
export const DocLockedSignal = () => (
  <span className="flex items-center gap-1.5 px-1 font-mono text-xs text-muted-foreground">
    <LockIcon className="size-3.5" aria-hidden />
    Locked
  </span>
);
