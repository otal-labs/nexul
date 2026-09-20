import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { CreateMemoryForm } from "@/components/memory/CreateMemoryForm";
import { MemoriesFeed } from "@/components/memory/MemoriesFeed";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { useFetchMemories } from "@/hooks/MemoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { emptyCreateMemoryForm, CreateMemoryFormSchema, type CreateMemoryFormData } from "@/models/Memory";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const MemoriesPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const canWrite = useHasPermission("memories:write");
  const { open: openCreateMemory } = useFormDialog();
  const { data: memories, error, isPending } = useFetchMemories(workspaceId);
  const { data: projects = [] } = useFetchProjects();

  const openCreateMemoryDialog = () =>
    openCreateMemory<CreateMemoryFormData>({
      title: "New memory",
      schema: CreateMemoryFormSchema,
      okLabel: "Create",
      form: <CreateMemoryForm />,
      formOptions: { defaultValues: emptyCreateMemoryForm() },
    });

  return (
    <Container className="p-6">
      <PageHeader
        className="mb-6"
        eyebrow="Memories"
        title="Memories"
        subtitle="Durable context for Agent, workspace-wide or per project — process and gotchas, never client requirements."
        actions={canWrite && <Button onClick={() => void openCreateMemoryDialog()}>New memory</Button>}
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {memories && memories.length === 0 && <NoDataDisplay message="No memories yet." />}
      {memories && memories.length > 0 && <MemoriesFeed memories={memories} projects={projects} />}
    </Container>
  );
};
