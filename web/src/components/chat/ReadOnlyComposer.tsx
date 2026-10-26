import { LockIcon } from "lucide-react";

import type { PermissionSource } from "@/models/Permission";

interface ReadOnlyComposerProps {
  label: string;
  source: PermissionSource;
}

// Where the composer sits in a channel the viewer reads but may not post in, with what decided it.
export const ReadOnlyComposer = ({ label, source }: ReadOnlyComposerProps) => (
  <div className="px-3 pt-1 pb-3">
    <div className="mx-auto flex w-full max-w-3xl items-start gap-3 rounded-lg border border-dashed border-input px-4 py-3">
      <LockIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 space-y-0.5">
        <p className="text-sm text-foreground">You can read {label} but not post in it.</p>
        <p className="text-xs text-pretty text-muted-foreground">{source.sentence}</p>
      </div>
    </div>
  </div>
);
