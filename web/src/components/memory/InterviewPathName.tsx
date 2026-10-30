import { cn } from "@/lib/utils";

interface InterviewPathNameProps {
  path: string;
  className?: string;
}

const TAIL = 10;

// Marks the head data-cut while its text runs past it: the mono ellipsis sits left in its cell, so the tail shifts left (a shift, not a margin, which would hand the folders the space).
const markCut = (el: HTMLSpanElement | null) => {
  if (!el) return;
  const observer = new ResizeObserver(() => el.toggleAttribute("data-cut", el.scrollWidth > el.clientWidth));
  observer.observe(el);
  return () => observer.disconnect();
};

// A path that gives way by the width it has: its folders go first, then the middle of its file name, which keeps its last
// characters and extension.
export const InterviewPathName = ({ path, className }: InterviewPathNameProps) => {
  const slash = path.lastIndexOf("/");
  const folders = path.slice(0, slash + 1);
  const name = path.slice(slash + 1);
  const dot = name.lastIndexOf(".");
  const stem = dot > 0 ? name.slice(0, dot) : name;
  const keep = Math.min(TAIL, Math.max(stem.length - 4, 0));
  const head = stem.slice(0, stem.length - keep);
  const tail = name.slice(head.length);
  return (
    <span title={path} className={cn("flex min-w-0 font-mono whitespace-nowrap", className)}>
      {folders !== "" && <span className="min-w-0 truncate">{folders}</span>}
      {/* Mono, so ch is exact: the head keeps its width until the folders are gone, then gives way down to the tail. */}
      <span
        ref={markCut}
        className="peer/head truncate"
        style={{ minWidth: `min(${head.length}ch, calc(100% - ${tail.length}ch))` }}>
        {head}
      </span>
      <span className="shrink-0 peer-data-cut/head:-translate-x-[0.4ch]">{tail}</span>
    </span>
  );
};
