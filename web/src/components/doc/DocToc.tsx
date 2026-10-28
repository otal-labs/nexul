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

// The section being read is the last heading above this share of the window's height.
const READING_LINE = 0.25;

// Which heading the reader is in. It looks the elements up fresh each time, since the editor replaces its nodes when a session syncs.
const headingInView = (headings: DocHeading[], scroller: Element | null): string | null => {
  const atEnd = scroller !== null && scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 2;
  const line = atEnd ? window.innerHeight : window.innerHeight * READING_LINE;
  let current: string | null = null;
  for (const heading of headings) {
    const el = findHeadingElement(heading);
    if (!el) continue;
    if (el.getBoundingClientRect().top > line) break;
    current = heading.id;
  }
  return current;
};

export const DocToc = ({ headings, onJump }: DocTocProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);

  useEffect(() => {
    if (headings.length < MIN_HEADINGS) return;
    let frame = 0;
    // Scroll doesn't bubble, so a capturing listener hears the pane's own scroll box, whichever element that is.
    const onScroll = (event: Event) => {
      if (frame) return;
      const scroller = event.target instanceof Element ? event.target : document.scrollingElement;
      // Only the box that scrolls the body moves its headings; the docs list or a popover scrolling says nothing.
      if (!scroller?.querySelector(".doc-body-view")) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        setActiveId(headingInView(headings, scroller));
      });
    };
    document.addEventListener("scroll", onScroll, { capture: true, passive: true });
    return () => {
      document.removeEventListener("scroll", onScroll, { capture: true });
      cancelAnimationFrame(frame);
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
      <ul className="mt-2.5 space-y-0.5 border-l border-border pl-1">
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
                // The edge is a bar laid on the track and entries start past it, so the rounded focus ring never crosses the track.
                "relative block rounded-md py-1 pr-2 text-[13px] leading-5 text-muted-foreground transition-colors duration-150 ease-standard before:absolute before:inset-y-1.5 before:-left-[5.5px] before:w-0.5 before:rounded-full hover:text-foreground",
                heading.level === topLevel && "pl-2.5",
                heading.level === topLevel + 1 && "pl-[1.125rem]",
                heading.level >= topLevel + 2 && "pl-[1.625rem]",
                currentId === heading.id && "font-medium text-foreground before:bg-brand",
              )}
            >
              <span className="block truncate">{heading.text}</span>
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
};
