import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryHistoryDialog } from "@/components/memory/MemoryHistoryDialog";
import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import { microheaderClass } from "@/components/Microheader";
import { useFetchHistory } from "@/hooks/HistoryHooks";
import { memoryHistoryUrl } from "@/hooks/MemoryHooks";

interface MemoryVersionsFeedProps {
  memoryId: string;
}

const SHOWN = 5;

// The kept versions, newest first as GET /api/memories/{id}/history answers them; a row opens the history dialog on it.
export const MemoryVersionsFeed = ({ memoryId }: MemoryVersionsFeedProps) => {
  const { data, error, isPending } = useFetchHistory(memoryHistoryUrl(memoryId));
  const [all, setAll] = useState(false);
  const [opened, setOpened] = useState<number>();
  const versions = data?.versions;
  const shown = all ? versions : versions?.slice(0, SHOWN);

  return (
    <section className="space-y-0.5" aria-label="Versions">
      <h2 className={`${microheaderClass} pb-1`}>
        Versions
        {versions && <span className="ml-1.5 tabular-nums">{versions.length}</span>}
      </h2>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {shown && (
        <EnterList className="-mx-2">
          {shown.map((version) => (
            <MemoryVersionRow key={version.version} version={version} onOpen={setOpened} />
          ))}
        </EnterList>
      )}
      {versions && !all && versions.length > SHOWN && (
        <button
          type="button"
          onClick={() => setAll(true)}
          className="rounded-sm pt-1 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          All {versions.length} versions
        </button>
      )}
      {opened !== undefined && <MemoryHistoryDialog memoryId={memoryId} version={opened} onClose={() => setOpened(undefined)} />}
    </section>
  );
};
