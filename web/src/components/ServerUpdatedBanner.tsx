import { RefreshCw, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useServerUpdateStore } from "@/stores/serverUpdateStore";

export const ServerUpdatedBanner = () => {
  const version = useServerUpdateStore((s) => s.pendingVersion);
  const dismiss = useServerUpdateStore((s) => s.dismiss);

  if (version === null) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-border bg-card px-3 py-2.5 duration-200 ease-out"
    >
      <RefreshCw className="size-4 shrink-0 text-info" aria-hidden />
      <p className="min-w-0 flex-1 basis-40 text-sm">
        Nexul was updated to <span className="font-mono tabular-nums">{version}</span>. Reload to use the new version.
      </p>
      <div className="flex items-center gap-1">
        <Button size="sm" onClick={() => window.location.reload()}>
          Reload
        </Button>
        <Button size="icon" variant="ghost" className="size-8" aria-label="Dismiss" title="Dismiss" onClick={dismiss}>
          <X className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  );
};
