import type { ReactNode } from "react";

import { RepoProviderMark } from "@/components/ProviderMarks";

interface WizardRepositoryCardProps {
  fullName: string;
  provider: string;
  meta: string;
  action: ReactNode;
}

// A repository the step already holds: what it is, one line on its role, and the one way to change it.
export const WizardRepositoryCard = ({ fullName, provider, meta, action }: WizardRepositoryCardProps) => (
  <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5">
    <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-surface-2 ring-1 ring-border ring-inset">
      <RepoProviderMark provider={provider} />
    </span>
    <span className="min-w-0 flex-1">
      <span className="block truncate text-sm font-medium">{fullName}</span>
      <span className="block truncate text-xs text-muted-foreground">{meta}</span>
    </span>
    {action}
  </div>
);
