import { useEffect } from "react";
import { useLocation } from "react-router";

import { findHeadingElement, type DocHeading } from "@/components/doc/docHeadings";

// A doc opened at #<heading id>, as a search hit links one, scrolls to that heading once the editor has drawn it.
export const useHashHeading = (headings: DocHeading[]) => {
  const { hash } = useLocation();
  const target = headings.find((heading) => `#${heading.id}` === hash);
  const id = target?.id;
  const text = target?.text;
  const level = target?.level;

  useEffect(() => {
    if (!id || text === undefined || level === undefined) return;
    const reveal = () => {
      const el = findHeadingElement({ id, text, level });
      el?.scrollIntoView?.({ block: "start" });
      return el !== null;
    };
    if (reveal()) return;
    const mutation = new MutationObserver(() => {
      if (reveal()) mutation.disconnect();
    });
    mutation.observe(document.body, { childList: true, subtree: true });
    return () => mutation.disconnect();
  }, [id, text, level]);
};
