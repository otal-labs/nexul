import { Suspense, useState } from "react";
import { useNavigate } from "react-router";

import { PermissionsForm, PermissionsFormSchema, type PermissionsFormData } from "@/components/access/PermissionsForm";
import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { LazyCreateDocForm } from "@/components/doc/LazyCreateDocForm";
import { DocsFeed } from "@/components/doc/DocsFeed";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDocs } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveDocFormSchema, type SaveDocFormData } from "@/models/Doc";
import { docPath, projectTokenById } from "@/models/Project";
import { emptyDocForm } from "@/utils/emptyDocJson";

export const DocsPage = () => {
  const navigate = useNavigate();
  const { open: openCreateDoc } = useFormDialog();
  const { open: openPermissions } = useFormDialog();
  const [selected, setSelected] = useState<string[]>([]);
  const { data, error, isPending } = useFetchDocs();
  const { data: projects = [] } = useFetchProjects();

  const openDoc = (id: string) => {
    const projectId = data?.find((d) => d.id === id)?.project_id;
    navigate(projectId ? docPath(projectTokenById(projects, projectId), id) : `/docs/${id}`);
  };

  const toggleSelect = (id: string) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((d) => d !== id) : [...prev, id]));
  };

  const openPermissionsDialog = async () => {
    await openPermissions<PermissionsFormData>({
      title: "Permissions",
      schema: PermissionsFormSchema,
      okLabel: "Apply",
      form: <PermissionsForm resourceType="doc" resourceIds={selected} />,
    });
    setSelected([]);
  };

  const openCreateDocDialog = () =>
    openCreateDoc<SaveDocFormData>({
      title: "New doc",
      schema: SaveDocFormSchema,
      okLabel: "Create",
      form: (
        <Suspense fallback={<LoadingDisplay />}>
          <LazyCreateDocForm />
        </Suspense>
      ),
      formOptions: { defaultValues: emptyDocForm() },
    });

  return (
    <Container className="p-6">
      <PageHeader
        className="mb-6"
        eyebrow="Docs"
        title="Docs"
        subtitle="The org&apos;s decisions, recorded — every doc is a source of truth that tickets can hang off."
      />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <DocsFeed
          docs={data}
          selected={selected}
          onToggleSelect={toggleSelect}
          onSelect={openDoc}
          onCreate={() => void openCreateDocDialog()}
          onPermissions={() => void openPermissionsDialog()}
        />
      )}
    </Container>
  );
};
