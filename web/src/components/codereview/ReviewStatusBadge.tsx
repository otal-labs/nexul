import { CircleAlert, CircleCheck, Clock, GitMerge, GitPullRequestClosed, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ReviewStatus, type ReviewStatus as ReviewStatusType } from "@/models/CodeReview";

// No-fill badge: merged/approved share success, changes-requested is the one warning, closed recedes to muted.
const iconByStatus: Record<ReviewStatusType, LucideIcon> = {
  [ReviewStatus.Pending]: Clock,
  [ReviewStatus.Approved]: CircleCheck,
  [ReviewStatus.ChangesRequested]: CircleAlert,
  [ReviewStatus.Merged]: GitMerge,
  [ReviewStatus.Closed]: GitPullRequestClosed,
};

const colorByStatus: Record<ReviewStatusType, string> = {
  [ReviewStatus.Pending]: "text-info",
  [ReviewStatus.Approved]: "text-success",
  [ReviewStatus.ChangesRequested]: "text-warning",
  [ReviewStatus.Merged]: "text-success",
  [ReviewStatus.Closed]: "text-muted-foreground",
};

interface ReviewStatusBadgeProps {
  status: ReviewStatusType;
}

export const ReviewStatusBadge = ({ status }: ReviewStatusBadgeProps) => (
  <NoFillBadge icon={iconByStatus[status]} color={colorByStatus[status]}>
    {status.replace("_", " ")}
  </NoFillBadge>
);
