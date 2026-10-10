import { useState } from "react";

import type { LiveSocket } from "@nexul/client-core/liveSocket";

import { DocThreadButton } from "@/components/chat/DocThreadButton";
import { DocActionsMenu } from "@/components/doc/DocActionsMenu";
import { DocAutoPlaysSection } from "@/components/doc/DocAutoPlaysSection";
import { DocBodySection } from "@/components/doc/DocBodySection";
import { DocLockedSignal } from "@/components/doc/DocLockedSignal";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { DocTitleField } from "@/components/doc/DocTitleField";
import { DocWatchButton } from "@/components/doc/DocWatchButton";
import { DocToc } from "@/components/doc/DocToc";
import { DocQuestionsPanel } from "@/components/doc/clarification/DocQuestionsPanel";
import { DocViewSwitch } from "@/components/doc/clarification/DocViewSwitch";
import { PointerOverlay } from "@/components/doc/collab/PointerOverlay";
import { useArticlePointer } from "@/components/doc/collab/useArticlePointer";
import { useCollabCommit } from "@/components/doc/collab/useCollabCommit";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { extractDocHeadings, type DocHeading } from "@/components/doc/docHeadings";
import { PageHeader } from "@/components/PageHeader";
import { PlaysMenu } from "@/components/play/PlaysMenu";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useDocCrumbs } from "@/hooks/useDocCrumbs";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import { useDocView } from "@/hooks/useDocView";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";
import type { Doc } from "@/models/Doc";
import type { MeResponse } from "@/models/User";

const collabIdentity = (me: MeResponse | undefined) => ({
  name: me?.user?.name ?? "",
  avatar: me?.user ? effectiveAvatar(me.user) : "",
});

interface DocDetailProps {
  doc: Doc;
  workspaceId: string;
  /** Omitted when the viewer may not create tickets, which hides the action. */
  onCreateTicket?: (() => void) | undefined;
  onPermissions: () => void;
  onArchive: () => void;
  onRestore: () => void;
  /** Test seam: the session's socket factory. */
  wsFactory?: (url: string) => LiveSocket;
}

export const DocDetail = ({ doc, workspaceId, onCreateTicket, onPermissions, onArchive, onRestore, wsFactory }: DocDetailProps) => {
  const token = useSessionStore((s) => s.token);
  const canThread = useHasPermission("docs:thread");
  const canWrite = useHasPermission("docs:write");
  // Display name for the collab presence comes from useFetchMe, not sessionStore (F5).
  const { data: me } = useFetchMe();
  const { name: userName, avatar: userAvatar } = collabIdentity(me);
  // A locked doc, or a reader the server would refuse an edit session, renders the title and body read-only.
  const session = useCollabSession(doc.locked || !canWrite ? undefined : doc.id, "edit", userName, token, {
    ...(wsFactory ? { wsFactory } : {}),
    ...(userAvatar ? { avatar: userAvatar } : {}),
  });
  const [headings, setHeadings] = useState<DocHeading[]>(() => extractDocHeadings(doc.body));
  const participants = session?.participants ?? [];
  const { articleRef, onPointerMove, onPointerLeave } = useArticlePointer(session);
  const { switchable, waiting, view, setView } = useDocView(doc);
  const crumbs = useEmbeddedCrumbs(useDocCrumbs(doc));

  const { title, titleInputRef, onTitleChange, confirmTitle, onBodyChange } = useCollabCommit(session, doc.title, doc.body);

  return (
    <div className="@container animate-in fade-in-0 slide-in-from-bottom-1 mx-auto w-full max-w-5xl duration-200 ease-out">
      <PageHeader
        crumbs={crumbs}
        title={
          <DocTitleField
            editable={!!session}
            title={title}
            staticTitle={doc.title}
            onChange={onTitleChange}
            onBlur={confirmTitle}
            inputRef={titleInputRef}
          />
        }
        meta={<DocPresenceBar participants={participants} connected={session?.connected} updatedAt={doc.updated_at} />}
        actions={
          <>
            {doc.locked && <DocLockedSignal docId={doc.id} />}
            <PlaysMenu workspaceId={workspaceId} projectId={doc.project_id} docId={doc.id} />
            <DocWatchButton docId={doc.id} />
            <DocThreadButton workspaceId={workspaceId} docId={doc.id} />
            <DocActionsMenu
              doc={doc}
              onCreateTicket={onCreateTicket}
              onPermissions={onPermissions}
              onArchive={onArchive}
              onRestore={onRestore}
            />
          </>
        }
      />
      <div className="mt-6 @3xl:flex @3xl:gap-8">
        {/* Hidden below @3xl; the has() rule keeps the column from reserving 14rem when neither section renders. */}
        <div className="hidden w-48 shrink-0 @3xl:has-[section]:block">
          <div className="sticky top-6 -mx-2 flex max-h-[calc(100vh-3rem)] flex-col gap-8 overflow-y-auto px-2">
            <DocToc headings={view === "doc" ? headings : []} />
            {canThread && <DocAutoPlaysSection docId={doc.id} className="-mx-2" />}
            {canThread && (
              <TrailSection
                workspaceId={workspaceId}
                targetType="doc"
                targetId={doc.id}
                rowLayout="stacked"
                className="-mx-2 border-t-0 pt-0"
              />
            )}
          </div>
        </div>

        <div className="min-w-0 flex-1 space-y-4">
          {switchable && <DocViewSwitch view={view} waiting={waiting} onChange={setView} />}
          {view === "questions" && <DocQuestionsPanel docId={doc.id} />}
          {/* Hidden, not unmounted, behind Questions so the editor keeps its session. */}
          <article
            ref={articleRef}
            hidden={view === "questions"}
            // A sheet of its own inside the panel, with page margins, so the doc reads as the thing being written.
            className="relative rounded-lg bg-card px-6 py-8 shadow-card ring-1 ring-border @3xl:px-12 @3xl:py-11"
            onPointerMove={onPointerMove}
            onPointerLeave={onPointerLeave}
          >
            {session && <PointerOverlay awareness={session.provider.awareness} selfID={session.doc.clientID} />}
            <DocBodySection
              doc={doc}
              session={session}
              onBodyChange={onBodyChange}
              onHeadingsChange={setHeadings}
            />
          </article>

          {/* The rail holds the trail from @3xl; both read the same query, so there is one request. */}
          {canThread && (
            <div className="space-y-4 @3xl:hidden">
              <DocAutoPlaysSection docId={doc.id} className="-mx-2" />
              <TrailSection workspaceId={workspaceId} targetType="doc" targetId={doc.id} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
