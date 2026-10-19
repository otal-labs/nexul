import { RepoProviderMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import type { RepoRef } from "@/models/Project";

interface WizardAttachedRepositoryRowProps {
  repo: RepoRef;
  keeping: boolean;
  onKeep: () => void;
}

// The repository the project already deploys from; keeping it ends setup's deploy path without picking and scanning it again.
export const WizardAttachedRepositoryRow = ({ repo, keeping, onKeep }: WizardAttachedRepositoryRowProps) => (
  <div className="relative flex items-center gap-3 bg-accent px-2 py-3">
    <span aria-hidden className="absolute inset-y-0 left-0 w-0.5 bg-brand" />
    <RepoProviderMark provider={repo.connector_id} />
    <span className="min-w-0 flex-1">
      <span className="block truncate text-sm font-medium">{repo.full_name}</span>
      <span className="block text-xs text-muted-foreground">Attached to this project. Pick another below to change it.</span>
    </span>
    <Button type="button" size="sm" onClick={onKeep} loading={keeping}>
      Keep it
    </Button>
  </div>
);
