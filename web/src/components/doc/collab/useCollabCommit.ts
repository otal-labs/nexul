import { useEffect, useRef, useState } from "react";

import type { CollabSession } from "@/components/doc/collab/useCollabSession";

// The title and body a live room commits; only the renaming client sends a title, so peers never stomp a rename.
export const useCollabCommit = (session: CollabSession | null, initialTitle: string, initialBody: string) => {
  const [title, setTitle] = useState(initialTitle);
  const titleRef = useRef(initialTitle);
  const bodyRef = useRef(initialBody);
  const titleDirtyRef = useRef(false);
  const lastCommittedTitleRef = useRef(initialTitle);
  const titleInputRef = useRef<HTMLTextAreaElement | null>(null);

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

  return {
    title,
    titleInputRef,
    onTitleChange: (value: string) => {
      setTitle(value);
      titleRef.current = value;
    },
    // Confirms on blur/Enter only — per-keystroke commits used to spam the updated event.
    confirmTitle: () => {
      if (!session || titleRef.current === lastCommittedTitleRef.current) return;
      titleDirtyRef.current = true;
      session.provider.markDirty();
      session.provider.flushCommit();
    },
    onBodyChange: (json: string) => {
      bodyRef.current = json;
    },
  };
};
