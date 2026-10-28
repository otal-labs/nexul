import { useState } from "react";

import type { LiveSocket } from "@nexul/client-core/liveSocket";

import { DocThreadButton } from "@/components/chat/DocThreadButton";
import { DocActionsMenu } from "@/components/doc/DocActionsMenu";
import { DocAutoPlaysSection } from "@/components/doc/DocAutoPlaysSection";
import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { DocBodySection } from "@/components/doc/DocBodySection";
import { DocLockedSignal } from "@/components/doc/DocLockedSignal";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { DocTicketsRunLine } from "@/components/doc/DocTicketsRunLine";
import { DocTitleField } from "@/components/doc/DocTitleField";
import { DocWatchButton } from "@/components/doc/DocWatchButton";
import { DocToc } from "@/components/doc/DocToc";
import { DocTocButton } from "@/components/doc/DocTocButton";
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
import { useActionState } from "@/hooks/useActionState";
import { useEntityActions } from "@/hooks/useEntityActions";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";
import type { Doc } from "@/models/Doc";
import type { MeResponse } from "@/models/User";

const collabIdentity = (me: MeResponse | undefined) => ({
  id: me?.user?.id ?? "",
  login: me?.user?.login ?? "",
  name: me?.user?.name ?? "",
  avatar: me?.user ? effectiveAvatar(me.user) : "",
});

interface DocDetailProps {
  doc: Doc;
  workspaceId: string;
  onCreateTicket: () => void;
  onPermissions: () => void;
  onArchive: () => void;
  onRestore: () => void;
  /** Test seam: the session's socket factory. */
  wsFactory?: (url: string) => LiveSocket;
}

export const DocDetail = ({ doc, workspaceId, onCreateTicket, onPermissions, onArchive, onRestore, wsFactory }: DocDetailProps) => {
  const token = useSessionStore((s) => s.token);
  const answer = useEntityActions("doc", doc.id);
  const canThread = useActionState({ permission: "docs:thread", projectId: doc.project_id, answer: answer("docs:thread") }).state === "live";
  const canWrite = useActionState({ permission: "docs:write", projectId: doc.project_id, answer: answer("docs:write") }).state === "live";
  // Display name for the collab presence comes from useFetchMe, not sessionStore (F5).
  const { data: me } = useFetchMe();
  const { id: userId, login: userLogin, name: userName, avatar: userAvatar } = collabIdentity(me);
  // A locked doc, or a reader the server would refuse an edit session, renders the title and body read-only.
  const session = useCollabSession(doc.locked || !canWrite ? undefined : doc.id, "edit", userName, token, {
    ...(wsFactory ? { wsFactory } : {}),
    ...(userId ? { userId } : {}),
    ...(userLogin ? { login: userLogin } : {}),
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
        actionsOn="crumbs"
        className="border-b-0 pb-0"
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
            {switchable && <DocViewSwitch view={view} waiting={waiting} onChange={setView} />}
            {switchable && <span aria-hidden className="h-4 w-px bg-border" />}
            {doc.locked && <DocLockedSignal docId={doc.id} />}
            {view === "doc" && <DocTocButton headings={headings} className="size-7 @min-[50rem]:hidden" />}
            <PlaysMenu workspaceId={workspaceId} projectId={doc.project_id} docId={doc.id} />
            <DocWatchButton docId={doc.id} />
            <DocThreadButton workspaceId={workspaceId} docId={doc.id} projectId={doc.project_id} />
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
      {/* The rail needs 50rem: a 35rem body, the gap and its 13rem; spare width sits between the body and the rail. */}
      <div className="mt-6 @min-[50rem]:flex @min-[50rem]:justify-between @min-[50rem]:gap-8">
        <div className="min-w-0 max-w-[42rem] flex-1 space-y-4">
          {canThread && <DocTicketsRunLine workspaceId={workspaceId} docId={doc.id} projectId={doc.project_id} />}
          {view === "questions" && <DocQuestionsPanel docId={doc.id} />}
          {/* Hidden, not unmounted, behind Questions so the editor keeps its session. */}
          <article
            ref={articleRef}
            hidden={view === "questions"}
            className="relative"
            onPointerMove={onPointerMove}
            onPointerLeave={onPointerLeave}
          >
            {session && <PointerOverlay awareness={session.provider.awareness} selfID={session.doc.clientID} />}
            <CalendarTitleContext value={doc.title}>
              <DocBodySection doc={doc} session={session} onBodyChange={onBodyChange} onHeadingsChange={setHeadings} />
            </CalendarTitleContext>
          </article>

          {/* The rail holds the trail once it is drawn; both read the same query, so there is one request. */}
          {canThread && (
            <div className="space-y-4 @min-[50rem]:hidden">
              <DocAutoPlaysSection workspaceId={workspaceId} docId={doc.id} className="-mx-2" />
              <TrailSection workspaceId={workspaceId} targetType="doc" targetId={doc.id} />
            </div>
          )}
        </div>

        {/* After the body, so the body keeps the title's edge; has() skips the rail when no section renders. */}
        <div className="hidden w-52 shrink-0 @min-[50rem]:has-[section]:block">
          <div className="sticky top-6 -mx-2 flex max-h-[calc(100vh-3rem)] flex-col gap-8 overflow-y-auto px-2">
            <DocToc headings={view === "doc" ? headings : []} />
            {canThread && <DocAutoPlaysSection workspaceId={workspaceId} docId={doc.id} className="-mx-2" />}
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
      </div>
    </div>
  );
};
