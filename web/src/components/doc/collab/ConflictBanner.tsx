import { AlertTriangleIcon } from "lucide-react";

import { Button } from "@/components/ui/button";

interface ConflictBannerProps {
  /** Keep my version: rebroadcast the full local state as the new truth. */
  onKeepMine: () => void;
  /** Take the server version: drop local state and replay the stored one. */
  onTakeServer: () => void;
}

// Fallback for unresolvable payloads — normal edits merge via Y.js; only apply failures land here.
export const ConflictBanner = ({ onKeepMine, onTakeServer }: ConflictBannerProps) => {
  return (
    <div
      role="alert"
      className="flex flex-wrap items-center gap-3 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm"
      data-testid="conflict-banner"
    >
      <AlertTriangleIcon className="size-4 text-destructive" aria-hidden="true" />
      <span className="min-w-0 flex-1 text-destructive">
        The session received an incompatible change and cannot merge it automatically.
      </span>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={onKeepMine}>
          Keep my version
        </Button>
        <Button variant="outline" size="sm" onClick={onTakeServer}>
          Use server version
        </Button>
      </div>
    </div>
  );
};
