import { GitPullRequest } from "lucide-react";
import { useMemo } from "react";

import { bodyToHtml } from "@/utils/RichtextUtility";
import { pullRequestLink } from "@/utils/TrailTranscriptUtility";

interface TrailReplyProseProps {
  text: string;
}

// The Agent's final reply as prose, through the same markdown pipeline docs render with, and a chip for the pull
// request it opened when the reply links one.
export const TrailReplyProse = ({ text }: TrailReplyProseProps) => {
  const html = useMemo(() => bodyToHtml(text), [text]);
  const pr = pullRequestLink(text);
  return (
    <div className="space-y-2 px-1 py-1">
      <div className="prose-rich text-sm" dangerouslySetInnerHTML={{ __html: html }} />
      {pr !== null && (
        <a
          href={pr.url}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 rounded-md border border-border bg-card px-2 py-1 font-mono text-[11px] transition-colors duration-150 ease-standard hover:bg-accent/40"
        >
          <GitPullRequest className="size-3.5" aria-hidden />
          {pr.label}
        </a>
      )}
    </div>
  );
};
