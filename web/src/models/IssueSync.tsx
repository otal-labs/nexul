export const IssueSyncMode = {
  Label: "label",
  All: "all",
} as const;

export type IssueSyncMode = (typeof IssueSyncMode)[keyof typeof IssueSyncMode];

// One repository's GitHub issue sync: labelled issues, or every issue, become the project's tickets and follow them.
export interface IssueSync {
  project_id: string;
  owner: string;
  repo: string;
  label: string;
  mode: IssueSyncMode;
  enabled: boolean;
  last_swept_at?: string;
}

export interface IssueSyncChange {
  owner: string;
  repo: string;
  enabled?: boolean;
  label?: string;
  mode?: IssueSyncMode;
}

export const issueSyncOf = (syncs: readonly IssueSync[] | undefined, owner: string, repo: string): IssueSync | undefined =>
  syncs?.find((s) => s.owner === owner && s.repo === repo);
