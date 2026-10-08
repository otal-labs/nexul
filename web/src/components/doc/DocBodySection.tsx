import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { CollabRichTextEditor } from "@/components/doc/collab/CollabRichTextEditor";
import type { CollabSession } from "@/components/doc/collab/useCollabSession";
import type { DocHeading } from "@/components/doc/docHeadings";
import type { Doc } from "@/models/Doc";

interface DocBodySectionProps {
  doc: Doc;
  session: CollabSession | null;
  onBodyChange: (json: string) => void;
  onHeadingsChange: (headings: DocHeading[]) => void;
}

export const DocBodySection = ({ doc, session, onBodyChange, onHeadingsChange }: DocBodySectionProps) => (
  <div className="mx-auto max-w-3xl [&_.tiptap>:first-child]:mt-0!">
    {!session && <DocBodyView body={doc.body} />}
    {session && (
      <CollabRichTextEditor
        session={session}
        value={doc.body}
        onChange={onBodyChange}
        onHeadingsChange={onHeadingsChange}
        aria-label="Body"
        attachTo={{ doc_id: doc.id }}
      />
    )}
    <AttachmentsSection owner={{ doc_id: doc.id }} className="mt-8" />
  </div>
);
