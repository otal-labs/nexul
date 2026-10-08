import { useParams } from "react-router";

import { PermissionsForm, PermissionsFormSchema, type PermissionsFormData } from "@/components/access/PermissionsForm";
import { Container } from "@/components/Container";
import { DocDetail } from "@/components/doc/DocDetail";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { CreateTicketFooter } from "@/components/ticket/CreateTicketFooter";
import { CreateTicketForm, emptyTicketForm } from "@/components/ticket/CreateTicketForm";
import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useArchiveDoc, useFetchDoc, useFetchDocClarification, useRestoreDoc } from "@/hooks/DocHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveTicketFormSchema, type SaveTicketFormData } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { LiveSocket } from "@/api/ws";

interface DocPageProps {
  /** Test seam: forwarded to the doc's collaboration session. */
  wsFactory?: (url: string) => LiveSocket;
  /** Overrides the route param — used to embed a doc's detail without navigating (e.g. Inbox's split view). */
  docId?: string;
}

export const DocPage = ({ wsFactory, docId: docIdProp }: DocPageProps = {}) => {
  const { docId: routeDocId } = useParams<{ docId: string }>();
  const docId = docIdProp ?? routeDocId;
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { open: openCreateTicket } = useFormDialog();
  const { open: openPermissions } = useFormDialog();
  const { data: doc, error, isPending } = useFetchDoc(docId);
  // Fetched beside the doc, so a page opening on its Questions view doesn't flash the article first.
  useFetchDocClarification(docId);
  const archiveDoc = useArchiveDoc();
  const restoreDoc = useRestoreDoc();

  const openCreateTicketDialog = (targetDocId: string) =>
    openCreateTicket<SaveTicketFormData>({
      title: "New ticket",
      schema: SaveTicketFormSchema,
      okLabel: "Create ticket",
      header: <ProjectDialogHeader title="New ticket" />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm docId={targetDocId} defaultProjectId={doc?.project_id ?? ""} />,
      formOptions: { defaultValues: emptyTicketForm() },
    });

  const openPermissionsDialog = (targetDocId: string) =>
    openPermissions<PermissionsFormData>({
      title: "Permissions",
      schema: PermissionsFormSchema,
      okLabel: "Apply",
      form: <PermissionsForm resourceType="doc" resourceIds={[targetDocId]} />,
    });

  return (
    <Container className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={docIdProp !== undefined} />}
      {doc && (
        <DocDetail
          doc={doc}
          workspaceId={workspaceId}
          {...(wsFactory ? { wsFactory } : {})}
          onCreateTicket={() => void openCreateTicketDialog(doc.id)}
          onPermissions={() => void openPermissionsDialog(doc.id)}
          onArchive={() => archiveDoc.mutate(doc.id)}
          onRestore={() => restoreDoc.mutate(doc.id)}
        />
      )}
    </Container>
  );
};
