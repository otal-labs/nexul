import { useState } from "react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import { microheaderClass } from "@/components/Microheader";
import { useFetchMemoryVersions } from "@/hooks/MemoryHooks";

interface MemoryVersionsFeedProps {
  memoryId: string;
  currentVersion: number;
}

const SHOWN = 5;

// Newest first, matching GET /api/memories/{id}/versions; the current version has no revert action.
export const MemoryVersionsFeed = ({ memoryId, currentVersion }: MemoryVersionsFeedProps) => {
  const { data, error, isPending } = useFetchMemoryVersions(memoryId);
  const [all, setAll] = useState(false);
  const shown = all ? data : data?.slice(0, SHOWN);

  return (
    <section className="space-y-0.5" aria-label="Versions">
      <h2 className={`${microheaderClass} pb-1`}>
        Versions
        {data && <span className="ml-1.5 tabular-nums">{data.length}</span>}
      </h2>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {shown && (
        <EnterList className="-mx-2">
          {shown.map((version) => (
            <MemoryVersionRow key={version.id} memoryId={memoryId} version={version} isCurrent={version.version === currentVersion} />
          ))}
        </EnterList>
      )}
      {data && !all && data.length > SHOWN && (
        <button
          type="button"
          onClick={() => setAll(true)}
          className="rounded-sm pt-1 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          All {data.length} versions
        </button>
      )}
    </section>
  );
};
