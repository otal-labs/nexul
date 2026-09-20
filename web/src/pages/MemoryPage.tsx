import { useParams, useNavigate } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoryDetail } from "@/components/memory/MemoryDetail";
import { useDeleteMemory, useFetchMemory, useUpdateMemory } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

interface MemoryPageProps {
  /** Overrides the route param — used to embed a memory's detail without navigating (e.g. Inbox's split view). */
  memoryId?: string;
}

export const MemoryPage = ({ memoryId: memoryIdProp }: MemoryPageProps = {}) => {
  const { memoryId: routeMemoryId } = useParams<{ memoryId: string }>();
  const memoryId = memoryIdProp ?? routeMemoryId;
  const navigate = useNavigate();
  const canWrite = useHasPermission("memories:write");
  const canDelete = useHasPermission("memories:delete");
  const canClone = useHasPermission("memories:clone");
  const { data: memory, error, isPending } = useFetchMemory(memoryId);
  const updateMemory = useUpdateMemory();
  const deleteMemory = useDeleteMemory();

  return (
    <Container className="p-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {memory && (
        <MemoryDetail
          memory={memory}
          canWrite={canWrite}
          canDelete={canDelete}
          canClone={canClone}
          saving={updateMemory.isPending}
          onSave={(input) => updateMemory.mutate({ id: memory.id, ...input })}
          onDelete={() => deleteMemory.mutate(memory.id, { onSuccess: () => navigate("/memories") })}
        />
      )}
    </Container>
  );
};
