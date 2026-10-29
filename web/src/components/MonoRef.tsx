import type { CSSProperties } from "react";

import { keptTail } from "@/utils/ImageRefUtility";
import { cn } from "@/lib/utils";

interface MonoRefProps {
  // An image by its last path segment (`imageShortName`), or a hostname.
  value: string;
  kind: "image" | "hostname";
  // The whole value, when what is shown is a shortened form of it.
  title?: string;
  className?: string;
  style?: CSSProperties;
}

// An identifier cut in its middle: the tail that names it (an image's tag, a hostname's domain) never shortens, and the
// head gives way on a whole mono character so its ellipsis sits flush against the tail. A copy still takes the whole text.
export const MonoRef = ({ value, kind, title, className, style }: MonoRefProps) => {
  const [head, tail] = keptTail(value, kind);
  // Half a pixel of slack on each side of the rounding: the container's width lands a hair under a whole character, and
  // an ellipsis a hair too wide drops one more.
  const headStyle: CSSProperties = { maxWidth: `calc(round(down, calc(100cqw - ${tail.length}ch + 0.5px), 1ch) + 0.5px)` };
  return (
    <span className={cn("@container block w-full min-w-0 font-mono", className)} style={style} title={title ?? value}>
      <span className="flex">
        <span className="shrink-0 truncate" style={headStyle}>{head}</span>
        {tail && <span className="shrink-0 whitespace-pre">{tail}</span>}
      </span>
    </span>
  );
};
