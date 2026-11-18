import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ImportHistoryCard } from "@/components/import/ImportHistoryCard";
import { ImportRunCard } from "@/components/import/ImportRunCard";
import { StartImportCard } from "@/components/import/StartImportCard";
import { useFetchImports } from "@/hooks/ImportHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { importOpen } from "@/models/Import";

// The workspace's one open import, or the form to start one, over the imports that have ended.
export const ImportSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: imports, error, isPending } = useFetchImports(workspaceId);
  const open = imports?.find(importOpen);
  const ended = imports?.filter((imp) => !importOpen(imp)) ?? [];

  return (
    <div className="space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {imports && open && <ImportRunCard id={open.id} />}
      {imports && !open && <StartImportCard />}
      {ended.length > 0 && <ImportHistoryCard imports={ended} />}
    </div>
  );
};
