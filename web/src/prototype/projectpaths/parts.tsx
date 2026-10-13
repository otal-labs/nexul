import type { ReactNode } from "react";

import { RepoProviderMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";

import { PICKED, SCAN_LINE } from "./mock";

export interface RepoStepProps {
  picked: boolean;
  onPick: () => void;
  onUnpick: () => void;
  onExit: (kind: "none" | "attached") => void;
}

export const StepHeading = ({ description, children }: { description: string; children?: ReactNode }) => (
  <div>
    <h2 className="text-base font-semibold tracking-tight">Repository</h2>
    <p className="mt-1 text-sm text-muted-foreground">{description}</p>
    {children}
  </div>
);

// The picked repository once its scan is back, standing in for the results list.
export const PickedSummary = ({ onChange }: { onChange: () => void }) => (
  <div className="flex items-center gap-3 border-y border-border px-2 py-3">
    <RepoProviderMark provider={PICKED.provider} />
    <span className="min-w-0 flex-1">
      <span className="block truncate text-sm font-medium">{PICKED.full_name}</span>
      <span className="block truncate font-mono text-xs text-muted-foreground">
        {PICKED.default_branch} · {SCAN_LINE}
      </span>
    </span>
    <Button type="button" variant="ghost" size="sm" onClick={onChange}>
      Change
    </Button>
  </div>
);

export const SectionLabel = ({ children }: { children: ReactNode }) => <p className="text-sm font-medium">{children}</p>;
