import { Link } from "react-router";
import { useEffect, useRef, useState } from "react";

import { DocThreadButton } from "@/components/chat/DocThreadButton";
import { DocActionsMenu } from "@/components/doc/DocActionsMenu";
import { DocBodySection } from "@/components/doc/DocBodySection";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { DocTitleField } from "@/components/doc/DocTitleField";
import { DocToc } from "@/components/doc/DocToc";
import { PointerOverlay } from "@/components/doc/collab/PointerOverlay";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { extractDocHeadings, type DocHeading } from "@/components/doc/docHeadings";
import { PlaysMenu } from "@/components/play/PlaysMenu";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { effectiveAvatar } from "@/models/User";
import { useSessionStore } from "@/stores/sessionStore";
import type { LiveSocket } from "@/api/ws";
import type { Doc } from "@/models/Doc";
import type { MeResponse } from "@/models/User";

const collabIdentity = (me: MeResponse | undefined) => ({
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

export const DocDetail = ({
  doc,
  workspaceId,
  onCreateTicket,
  onPermissions,
  onArchive,
  onRestore,
  wsFactory,
}: DocDetailProps) => {
  const token = useSessionStore((s) => s.token);
  const canThread = useHasPermission("docs:thread");
  // Display name for the collab presence comes from useFetchMe, not sessionStore (F5).
  const { data: me } = useFetchMe();
  const { name: userName, avatar: userAvatar } = collabIdentity(me);
  const session = useCollabSession(doc.id, "edit", userName, token, {
    ...(wsFactory ? { wsFactory } : {}),
    ...(userAvatar ? { avatar: userAvatar } : {}),
  });
  const [headings, setHeadings] = useState<DocHeading[]>(() => extractDocHeadings(doc.body));
  const participants = session?.participants ?? [];
  const [title, setTitle] = useState(doc.title);
  const titleRef = useRef(doc.title);
  const bodyRef = useRef(doc.body);
  // Only the renaming client sends a title; empty title on the wire means "unchanged".
  const titleDirtyRef = useRef(false);
  const lastCommittedTitleRef = useRef(doc.title);
  const titleInputRef = useRef<HTMLInputElement | null>(null);
  const articleRef = useRef<HTMLElement | null>(null);
  const lastPointerSent = useRef(0);

  // Refs skip a re-render per keystroke; reading title clears dirty to avoid stomping peer renames.
  useEffect(() => {
    if (!session) return;
    session.setGetState(() => {
      const dirty = titleDirtyRef.current;
      titleDirtyRef.current = false;
      if (dirty) lastCommittedTitleRef.current = titleRef.current;
      return { title: dirty ? titleRef.current : "", body: bodyRef.current };
    });
    session.setOnRemoteTitle((remote) => {
      lastCommittedTitleRef.current = remote;
      // A peer renamed. Their title wins unless this input is mid-edit.
      if (document.activeElement === titleInputRef.current) return;
      setTitle(remote);
      titleRef.current = remote;
      titleDirtyRef.current = false;
    });
    return () => {
      // Mark dirty on unmount so an unconfirmed rename isn't lost when navigating away.
      if (titleRef.current !== lastCommittedTitleRef.current) {
        titleDirtyRef.current = true;
        session.provider.markDirty();
      }
    };
  }, [session]);

  // Confirms on blur/Enter only — per-keystroke commits used to spam doc.updated.
  const confirmTitle = () => {
    if (!session || titleRef.current === lastCommittedTitleRef.current) return;
    titleDirtyRef.current = true;
    session.provider.markDirty();
    session.provider.flushCommit();
  };

  return (
    <div className="animate-in fade-in-0 slide-in-from-bottom-1 relative mx-auto w-full max-w-6xl duration-200 ease-out">
      <div className="flex items-center justify-between">
        <Link
          to="/docs"
          className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          ← All docs
        </Link>
        <div className="flex items-center gap-1">
          <PlaysMenu workspaceId={workspaceId} projectId={doc.project_id} docId={doc.id} />
          <DocThreadButton workspaceId={workspaceId} docId={doc.id} />
          <DocActionsMenu
            doc={doc}
            onCreateTicket={onCreateTicket}
            onPermissions={onPermissions}
            onArchive={onArchive}
            onRestore={onRestore}
          />
        </div>
      </div>

      {headings.length > 0 && (
        <div className="pointer-events-none absolute top-16 right-full bottom-0 hidden w-56 xl:block">
          <div className="pointer-events-auto sticky top-6 mr-8">
            <DocToc headings={headings} />
          </div>
        </div>
      )}

      <article
        ref={articleRef}
        className="relative mt-4 rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10 lg:p-14"
        onPointerMove={(e) => {
          if (!session) return;
          // ~20Hz is plenty for a presence pointer and keeps awareness frames cheap.
          const now = performance.now();
          if (now - lastPointerSent.current < 50) return;
          lastPointerSent.current = now;
          const rect = articleRef.current?.getBoundingClientRect();
          if (!rect) return;
          session.provider.awareness.setLocalStateField("pointer", {
            x: Math.round(e.clientX - rect.left),
            y: Math.round(e.clientY - rect.top),
          });
        }}
        onPointerLeave={() => session?.provider.awareness.setLocalStateField("pointer", null)}
      >
        {session && <PointerOverlay awareness={session.provider.awareness} selfID={session.doc.clientID} />}
        <DocPresenceBar participants={participants} connected={session?.connected} updatedAt={doc.updated_at} />
        <DocTitleField
          editable={!!session}
          title={title}
          staticTitle={doc.title}
          onChange={(value) => {
            setTitle(value);
            titleRef.current = value;
          }}
          onBlur={confirmTitle}
          inputRef={titleInputRef}
        />

        <DocBodySection
          doc={doc}
          session={session}
          onBodyChange={(json) => {
            bodyRef.current = json;
          }}
          onHeadingsChange={setHeadings}
        />
      </article>

      {canThread && <TrailSection workspaceId={workspaceId} targetType="doc" targetId={doc.id} />}
    </div>
  );
};
