export const ReviewStatus = {
  Pending: "pending",
  Approved: "approved",
  ChangesRequested: "changes_requested",
  Merged: "merged",
  Closed: "closed",
} as const;

export type ReviewStatus = (typeof ReviewStatus)[keyof typeof ReviewStatus];

export interface CodeReview {
  id: string;
  pr_number: number;
  repo: string;
  status: ReviewStatus;
  reviewer: string;
  created_at: string;
}
