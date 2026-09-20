import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { ConflictBanner } from "@/components/doc/collab/ConflictBanner";
import type { CollabSession } from "@/components/doc/collab/useCollabSession";
import type { DocHeading } from "@/components/doc/docHeadings";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import type { Doc } from "@/models/Doc";

interface DocBodySectionProps {
  doc: Doc;
  session: CollabSession | null;
  onBodyChange: (json: string) => void;
  onHeadingsChange: (headings: DocHeading[]) => void;
}

export const DocBodySection = ({ doc, session, onBodyChange, onHeadingsChange }: DocBodySectionProps) => (
  <div className="mt-7">
    {!session && <DocBodyView body={doc.body} />}
    {session && (
      <>
        <RichTextEditor
          value={doc.body}
          onChange={onBodyChange}
          onHeadingsChange={onHeadingsChange}
          aria-label="Body"
          attachTo={{ doc_id: doc.id }}
          collab={{
            doc: session.doc,
            provider: session.provider,
            user: session.user,
            serverReady: session.serverReady,
            hasServerState: session.hasServerState,
          }}
        />
        {session.applyError && (
          <ConflictBanner
            onKeepMine={() => {
              session.dismissApplyError();
              session.provider.recoverKeepMine();
            }}
            onTakeServer={() => {
              session.dismissApplyError();
              session.resetSession();
            }}
          />
        )}
      </>
    )}
    <AttachmentsSection owner={{ doc_id: doc.id }} className="mt-8" />
  </div>
);
