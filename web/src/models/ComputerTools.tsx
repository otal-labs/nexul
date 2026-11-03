import type { AxiosError } from "axios";

import type { ApiErrorBody } from "@/api/client";
import type { ToolFacts } from "@/models/ComputerFacts";

// The git name and email a person's GitHub sign-in gives a computer: their GitHub name and its no-reply address.
export interface GitIdentity {
  name: string;
  email: string;
}

export type ToolName = ToolFacts["name"];

// The tool a failed request says is missing on a computer, from the error envelope's details; undefined for any other failure.
export const missingToolOf = (error: unknown): { tool: ToolName; computerId: string } | undefined => {
  const details = (error as AxiosError<ApiErrorBody> | null)?.response?.data?.details as { missing_tool?: unknown; computer_id?: unknown } | undefined;
  const tool = details?.missing_tool;
  if ((tool !== "git" && tool !== "gh") || typeof details?.computer_id !== "string") return undefined;
  return { tool, computerId: details.computer_id };
};

// A git identity as git writes it in a commit.
export const gitAuthor = (identity: { name?: string | undefined; email?: string | undefined }) => [identity.name, identity.email && `<${identity.email}>`].filter(Boolean).join(" ");
