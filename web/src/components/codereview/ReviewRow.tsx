import { GitPullRequestIcon } from "lucide-react";

import { ReviewStatusBadge } from "@/components/codereview/ReviewStatusBadge";
import { ReviewerAvatars } from "@/components/codereview/ReviewerAvatars";
import { cn } from "@/lib/utils";
import type { CodeReview } from "@/models/CodeReview";

interface ReviewRowProps {
  review: CodeReview;
  /** Position within the list — drives the first-8 mount stagger. */
  index?: number;
}

// Row pattern: hairline `divide-y`, background-lift-only hover, mono/tabular-nums for the id.
export const ReviewRow = ({ review, index }: ReviewRowProps) => {
  const staggered = index != null && index < 8;

  return (
    <li
      className={cn(
        "flex items-center gap-3 px-4 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40",
        staggered && "animate-in fade-in-0 slide-in-from-bottom-1 duration-150 ease-out",
      )}
      style={staggered ? { animationDelay: `${index! * 25}ms`, animationFillMode: "both" } : undefined}
    >
      <GitPullRequestIcon className="size-4 shrink-0 text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate font-mono text-sm tabular-nums">
        {review.repo}#{review.pr_number}
      </span>
      {review.reviewer && (
        <span className="flex items-center gap-1.5">
          <ReviewerAvatars names={[review.reviewer]} />
          <span className="font-mono text-xs text-muted-foreground">by {review.reviewer}</span>
        </span>
      )}
      <ReviewStatusBadge status={review.status} />
    </li>
  );
};
