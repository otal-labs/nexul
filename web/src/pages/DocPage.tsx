import { useParams } from "react-router";

import type { LiveSocket } from "@nexul/client-core/liveSocket";

import { Container } from "@/components/Container";
import { DocDetail } from "@/components/doc/DocDetail";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useArchiveDoc, useFetchDoc, useFetchDocClarification, useRestoreDoc } from "@/hooks/DocHooks";
import { useCreateTicketDialog } from "@/hooks/useCreateTicketDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";

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
  const openPermissions = usePermissionsDialog();
  const { data: doc, error, isPending } = useFetchDoc(docId);
  const createTicket = useCreateTicketDialog(doc?.project_id ?? "");
  // Fetched beside the doc, so a page opening on its Questions view doesn't flash the article first.
  useFetchDocClarification(docId);
  const archiveDoc = useArchiveDoc();
  const restoreDoc = useRestoreDoc();

  return (
    <Container className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} embedded={docIdProp !== undefined} />}
      {doc && (
        <DocDetail
          doc={doc}
          workspaceId={workspaceId}
          {...(wsFactory ? { wsFactory } : {})}
          onCreateTicket={() => void createTicket?.({ docId: doc.id })}
          onPermissions={() =>
            void openPermissions({ type: "doc", id: doc.id, name: doc.title, workspaceId, projectId: doc.project_id, folderId: doc.folder_id })
          }
          onArchive={() => archiveDoc.mutate(doc.id)}
          onRestore={() => restoreDoc.mutate(doc.id)}
        />
      )}
    </Container>
  );
};
