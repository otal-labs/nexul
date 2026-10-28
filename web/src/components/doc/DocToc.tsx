import { useEffect, useState } from "react";

import type { DocHeading } from "@/components/doc/docHeadings";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

// A single heading is the title; a list of one is a column spent on nothing.
const MIN_HEADINGS = 2;

interface DocTocProps {
  headings: DocHeading[];
  /** Called after a jump, so a popover holding the outline can close. */
  onJump?: () => void;
}

// Matches by text+level, not anchor ids, since the body view mounts its own Tiptap instance.
function findHeadingElement(heading: DocHeading): HTMLElement | null {
  const els = document.querySelectorAll<HTMLElement>(
    ".doc-body-view h1, .doc-body-view h2, .doc-body-view h3, .doc-body-view h4, .doc-body-view h5, .doc-body-view h6",
  );
  for (const el of els) {
    if (el.textContent === heading.text && el.tagName === `H${heading.level}`) {
      if (!el.dataset.docHeadingId) el.dataset.docHeadingId = heading.id;
      return el;
    }
  }
  return null;
}

// Spy attaches once heading elements exist (MutationObserver) since the editor renders async.
export const DocToc = ({ headings, onJump }: DocTocProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);

  useEffect(() => {
    if (headings.length < MIN_HEADINGS) return;
    let observer: IntersectionObserver | null = null;
    let mutation: MutationObserver | null = null;

    const attach = () => {
      const els = headings
        .map(findHeadingElement)
        .filter((el): el is HTMLElement => el !== null);
      if (els.length === 0) return false;
      if (typeof IntersectionObserver === "undefined") return true;
      observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting && entry.target instanceof HTMLElement) {
              const id = entry.target.dataset.docHeadingId;
              if (id) setActiveId(id);
            }
          }
        },
        { rootMargin: "-20% 0px -70% 0px" },
      );
      for (const el of els) observer.observe(el);
      return true;
    };

    if (!attach()) {
      mutation = new MutationObserver(() => {
        if (attach()) mutation?.disconnect();
      });
      mutation.observe(document.body, { childList: true, subtree: true });
    }
    return () => {
      observer?.disconnect();
      mutation?.disconnect();
    };
  }, [headings]);

  // Before the reader scrolls, the first heading is where they are.
  const currentId = activeId ?? headings[0]?.id;
  // Indents count from the doc's top heading level, so a doc that starts at h2 doesn't open one step in.
  const topLevel = Math.min(...headings.map((heading) => heading.level));

  const jump = (heading: DocHeading) => {
    setActiveId(heading.id);
    onJump?.();
    const el = findHeadingElement(heading);
    if (!el) return;
    const reduceMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    el.scrollIntoView?.({ behavior: reduceMotion ? "auto" : "smooth", block: "start" });
  };

  if (headings.length < MIN_HEADINGS) return null;
  return (
    <section aria-label="On this page">
      <h2 className={microheaderClass}>
        On this page
      </h2>
      <ul className="mt-2.5 space-y-0.5 border-l border-border">
        {headings.map((heading) => (
          <li key={heading.id}>
            <a
              href={`#${heading.id}`}
              onClick={(event) => {
                event.preventDefault();
                jump(heading);
              }}
              title={heading.text}
              aria-current={currentId === heading.id ? "true" : undefined}
              className={cn(
                "-ml-px block truncate border-l-2 border-transparent py-1 pr-2 text-[13px] text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground",
                heading.level === topLevel && "pl-3",
                heading.level === topLevel + 1 && "pl-5",
                heading.level >= topLevel + 2 && "pl-7",
                currentId === heading.id && "border-brand font-medium text-foreground",
              )}
            >
              {heading.text}
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
};
