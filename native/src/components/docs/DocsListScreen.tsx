import { useRouter } from "expo-router";
import { ChevronsUpDown, FileText } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { DocsFeed } from "@/components/docs/DocsFeed";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { DocListItem } from "@/models/Doc";
import { effectiveProject } from "@/models/Project";
import { useDocsProjectStore } from "@/stores/docsProjectStore";

export const DocsListScreen = () => {
  const router = useRouter();
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const selectedProjectId = useDocsProjectStore((s) => s.selectedProjectId);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchProjects();
  const activeProject = effectiveProject(projects, selectedProjectId);
  const activeProjectId = activeProject?.id;
  const { data: docs, error: docsError, isPending: docsPending, isRefetching, refetch } = useFetchDocsByProject(activeProjectId);

  const onSelect = (doc: DocListItem) => {
    if (doc.can_open) router.push(`/more/docs/${doc.id}`);
  };

  const header = (
    <ScreenHeader
      eyebrow={activeProject?.name}
      title="Docs"
      meta={docs && docs.length > 0 ? `${docs.length} ${docs.length === 1 ? "doc" : "docs"}` : undefined}
      className="pt-2"
      action={
        projects &&
        projects.length > 1 && (
          <Pressable
            role="button"
            aria-label="Change project"
            onPress={() => router.push("/more/docs/pick-project")}
            className="size-11 items-center justify-center rounded-md border border-input active:bg-accent"
          >
            <ChevronsUpDown size={18} color={String(muted)} />
          </Pressable>
        )
      }
    />
  );
  const listed = activeProjectId && docs && docs.length > 0;

  return (
    <View className="flex-1 bg-background">
      {!listed && header}
      {projectsPending && <LoadingDisplay message="Loading projects" />}
      {projectsError && <ErrorDisplay error={projectsError} />}
      {projects && projects.length === 0 && <EmptyState icon={FileText} title="No projects yet" message="Docs live in a project; projects are created on the web." />}
      {activeProjectId && docsPending && <LoadingDisplay message="Loading docs" />}
      {activeProjectId && docsError && <ErrorDisplay error={docsError} />}
      {activeProjectId && docs && docs.length === 0 && <EmptyState icon={FileText} title="No docs yet" message="Docs written on the web for this project show up here." />}
      {listed && <DocsFeed docs={docs} header={header} refreshing={isRefetching} onRefresh={() => void refetch()} onSelect={onSelect} />}
    </View>
  );
};
