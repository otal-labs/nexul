import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryDetail } from "@/components/memory/MemoryDetail";
import { useFetchMemory, useUpdateMemory } from "@/hooks/MemoryHooks";
import { useConfirmDeleteMemory } from "@/hooks/useConfirmDeleteMemory";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

interface MemoryPageProps {
  /** Overrides the route param — used to embed a memory's detail without navigating (e.g. Inbox's split view). */
  memoryId?: string;
}

export const MemoryPage = ({ memoryId: memoryIdProp }: MemoryPageProps = {}) => {
  const { memoryId: routeMemoryId } = useParams<{ memoryId: string }>();
  const memoryId = memoryIdProp ?? routeMemoryId;
  const canWrite = useHasPermission("memories:write");
  const canDelete = useHasPermission("memories:delete");
  const canClone = useHasPermission("memories:clone");
  const { data: memory, error, isPending } = useFetchMemory(memoryId);
  const updateMemory = useUpdateMemory();
  const confirmDelete = useConfirmDeleteMemory();

  return (
    <Container className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={memoryIdProp !== undefined} />}
      {memory && (
        <MemoryDetail
          memory={memory}
          canWrite={canWrite}
          canDelete={canDelete}
          canClone={canClone}
          saving={updateMemory.isPending}
          onSave={(input) => updateMemory.mutate({ id: memory.id, always_included: memory.always_included, ...input })}
          onDelete={() => void confirmDelete(memory, true)}
        />
      )}
    </Container>
  );
};
