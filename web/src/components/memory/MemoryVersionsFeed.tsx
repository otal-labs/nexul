import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryVersionRow } from "@/components/memory/MemoryVersionRow";
import { useFetchMemoryVersions } from "@/hooks/MemoryHooks";

interface MemoryVersionsFeedProps {
  memoryId: string;
  currentVersion: number;
  canRevert: boolean;
}

// Newest first, matching GET /api/memories/{id}/versions; the current version has no revert action.
export const MemoryVersionsFeed = ({ memoryId, currentVersion, canRevert }: MemoryVersionsFeedProps) => {
  const { data, error, isPending } = useFetchMemoryVersions(memoryId);

  return (
    <section className="mt-8 space-y-3" aria-label="Version history">
      <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
        Version history
      </h2>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {data.map((version) => (
            <MemoryVersionRow
              key={version.id}
              memoryId={memoryId}
              version={version}
              isCurrent={version.version === currentVersion}
              canRevert={canRevert}
            />
          ))}
        </ul>
      )}
    </section>
  );
};
