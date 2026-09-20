import { z } from "zod";

export const TicketStatus = {
  Open: "open",
  InProgress: "in_progress",
  Done: "done",
  Closed: "closed",
} as const;

export type TicketStatus = (typeof TicketStatus)[keyof typeof TicketStatus];

export interface Ticket {
  id: string;
  project_id: string;
  category_id: string;
  type_id: string;
  title: string;
  body: string;
  status: TicketStatus;
  // Scoped to the ticket's current (status, category_id) pair; resets to the end of the pair on a move.
  position: number;
  // Per-project sequential number, combined with the project's prefix for a human-readable PREFIX-N id.
  number: number;
  doc_id: string;
  assignee: string;
  created_at: string;
  updated_at: string;
  finished_at?: string | null;
  // The gateway serializes labels as null when a ticket has none, so the type must say so.
  labels: string[] | null;
}

// Links use the human-readable PREFIX-NUMBER key, falling back to the UUID when the prefix isn't at hand.
export const ticketPath = (ticket: Ticket, prefix?: string): string =>
  `/tickets/${prefix ? `${prefix}-${ticket.number}` : ticket.id}`;

// A UUID can't false-match here: its first segment is 8 hex chars, longer than any prefix.
export const parseTicketKey = (param: string): { prefix: string; number: number } | undefined => {
  const [, prefix, number] = /^([A-Za-z]{2,5})-(\d+)$/.exec(param) ?? [];
  if (!prefix || !number) return undefined;
  return { prefix, number: Number(number) };
};

export const PRLinkState = {
  Open: "open",
  Merged: "merged",
  Closed: "closed",
} as const;

export type PRLinkState = (typeof PRLinkState)[keyof typeof PRLinkState];

export interface PRLink {
  owner: string;
  repo: string;
  number: number;
  title: string;
  sha: string;
  state: PRLinkState;
}

export interface BranchLink {
  owner: string;
  repo: string;
  branch: string;
}

export interface TicketLinks {
  prs: PRLink[];
  branches: BranchLink[];
}

export const LinkPRFormSchema = z.object({
  owner: z.string().min(1, "Owner is required"),
  repo: z.string().min(1, "Repository is required"),
  number: z.coerce.number<number>().int().positive("PR number must be a positive integer"),
});

export type LinkPRFormData = z.infer<typeof LinkPRFormSchema>;

export const LinkBranchFormSchema = z.object({
  owner: z.string().min(1, "Owner is required"),
  repo: z.string().min(1, "Repository is required"),
  branch: z.string().min(1, "Branch name is required"),
});

export type LinkBranchFormData = z.infer<typeof LinkBranchFormSchema>;

export const SaveTicketFormSchema = z.object({
  title: z.string().min(1, "Title is required"),
  body: z.string(),
  project_id: z.string().min(1, "A project is required"),
  doc_id: z.string().optional(),
  assignee: z.string().optional(),
  category_id: z.string().optional(),
  type_id: z.string().optional(),
});

export type SaveTicketFormData = z.infer<typeof SaveTicketFormSchema>;
