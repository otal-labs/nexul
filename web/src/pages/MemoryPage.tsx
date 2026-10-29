import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryDetail } from "@/components/memory/MemoryDetail";
import { useFetchMemory, useUpdateMemory } from "@/hooks/MemoryHooks";
import { useConfirmDeleteMemory } from "@/hooks/useConfirmDeleteMemory";
import { cn } from "@/lib/utils";

interface MemoryPageProps {
  /** Overrides the route param — used to embed a memory's detail without navigating (e.g. Inbox's split view). */
  memoryId?: string;
}

export const MemoryPage = ({ memoryId: memoryIdProp }: MemoryPageProps = {}) => {
  const { memoryId: routeMemoryId } = useParams<{ memoryId: string }>();
  const memoryId = memoryIdProp ?? routeMemoryId;
  const { data: memory, error, isPending } = useFetchMemory(memoryId);
  const updateMemory = useUpdateMemory();
  const confirmDelete = useConfirmDeleteMemory();

  return (
    // ponytail: the 10px is the thin scrollbar on Linux and Windows; an overlay scrollbar (macOS) leaves the right inset 10px short.
    <Container className={cn("py-8", memoryIdProp === undefined && "sm:pr-3.5")}>
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={memoryIdProp !== undefined} />}
      {memory && (
        <MemoryDetail
          memory={memory}
          saving={updateMemory.isPending}
          onSave={(input) => updateMemory.mutate({ id: memory.id, always_included: memory.always_included, ...input })}
          onDelete={() => void confirmDelete(memory, true)}
        />
      )}
    </Container>
  );
};
