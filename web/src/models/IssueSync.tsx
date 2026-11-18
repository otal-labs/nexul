// A repository's GitHub issue sync: in label mode the issues carrying the label become tickets, in all mode every one.
export interface IssueSync {
  project_id: string;
  owner: string;
  repo: string;
  label: string;
  mode: "label" | "all";
  enabled: boolean;
  last_swept_at?: string;
}

export const syncOf = (syncs: IssueSync[] | undefined, owner: string, repo: string): IssueSync | undefined =>
  syncs?.find((s) => s.owner === owner && s.repo === repo);
