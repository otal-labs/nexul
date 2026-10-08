import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import { microheaderClass } from "@/components/Microheader";
import { useFetchMemoryVersions } from "@/hooks/MemoryHooks";
import { cn } from "@/lib/utils";

interface MemoryVersionsFeedProps {
  memoryId: string;
  currentVersion: number;
  canRevert: boolean;
  /** Passed to each row; "stacked" suits a narrow column. */
  rowLayout?: "inline" | "stacked";
}

// Newest first, matching GET /api/memories/{id}/versions; the current version has no revert action.
export const MemoryVersionsFeed = ({ memoryId, currentVersion, canRevert, rowLayout = "inline" }: MemoryVersionsFeedProps) => {
  const { data, error, isPending } = useFetchMemoryVersions(memoryId);
  const stacked = rowLayout === "stacked";

  return (
    <section className={stacked ? "space-y-0.5" : "space-y-3"} aria-label="Versions">
      <h2 className={cn(microheaderClass, stacked && "pb-1")}>
        Versions
      </h2>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <ul className={cn(stacked ? "-mx-2" : "divide-y divide-border overflow-hidden rounded-md border")}>
          {data.map((version) => (
            <MemoryVersionRow
              key={version.id}
              memoryId={memoryId}
              version={version}
              isCurrent={version.version === currentVersion}
              canRevert={canRevert}
              layout={rowLayout}
            />
          ))}
        </ul>
      )}
    </section>
  );
};
