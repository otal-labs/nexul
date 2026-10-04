import type { ComponentProps } from "react";

import { ConflictBanner } from "@/components/doc/collab/ConflictBanner";
import type { CollabSession } from "@/components/doc/collab/useCollabSession";
import { RichTextEditor } from "@/components/doc/RichTextEditor";

interface CollabRichTextEditorProps
  extends Pick<ComponentProps<typeof RichTextEditor>, "value" | "onChange" | "onHeadingsChange" | "aria-label" | "attachTo"> {
  session: CollabSession;
}

// A body joined to its live room, with the recovery banner for an update the room could not apply.
export const CollabRichTextEditor = ({ session, ...editor }: CollabRichTextEditorProps) => (
  <>
    <RichTextEditor
      key={session.key}
      {...editor}
      collab={{ doc: session.doc, provider: session.provider, user: session.user, seed: session.seed }}
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
);
