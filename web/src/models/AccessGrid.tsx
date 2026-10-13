// Mirrors internal/access/grid.go's Grid: a workspace's Access page as the viewer may manage it.
export interface AccessGridRole {
  id: string;
  name: string;
  holders: number;
}

export interface AccessGridProject {
  id: string;
  name: string;
  prefix: string;
  role_ids: string[];
  user_ids: string[];
}

export interface AccessGridChannel {
  id: string;
  name: string;
  kind: "channel" | "voice_channel";
  hidden_role_ids: string[];
}

export interface AccessSetup {
  finished: boolean;
  finished_by?: string;
  finished_by_login?: string;
  finished_at?: string;
}

export interface AccessGrid {
  workspace_id: string;
  setup: AccessSetup;
  roles: AccessGridRole[];
  projects: AccessGridProject[];
  channels: AccessGridChannel[];
}

// Who each named project lists and which roles each named channel is hidden from; rows not named are left alone.
export interface AccessGridSave {
  projects?: { project_id: string; role_ids: string[]; user_ids: string[] }[];
  channels?: { channel_id: string; hidden_role_ids: string[] }[];
  finish?: boolean;
}

// The lead row's box: ticked when every row has the role, a dash when some do.
export const leadState = (count: number, rows: number): boolean | "indeterminate" => {
  if (count === 0) return false;
  if (count === rows) return true;
  return "indeterminate";
};

export const countLabel = (n: number, one: string, many: string): string => `${n} ${n === 1 ? one : many}`;
