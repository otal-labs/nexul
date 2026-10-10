import type { AxiosError } from "axios";

import type { ApiErrorBody } from "@/api/client";

// Whether the person's own GitHub sign-in lists their repositories (ADR 0147).
export type GitHubLinkState = "connected" | "reconnect" | "none";

export interface GitHubLinkStatus {
  state: GitHubLinkState;
  // The GitHub account the link reads as.
  login?: string;
  connected_at?: string;
}

type GitHubLinkRefusal = Exclude<GitHubLinkState, "connected">;

const refusalStates: Record<string, GitHubLinkRefusal> = { github_not_connected: "none", github_reconnect: "reconnect" };

// A repository read refused because the person's GitHub sign-in can't list their repositories, as the state to fix.
export const githubLinkRefusal = (error: unknown): GitHubLinkRefusal | undefined =>
  refusalStates[(error as AxiosError<ApiErrorBody> | null)?.response?.data?.code ?? ""];
