import type { AutomationVersionStatus } from "@/enums/Automation";

// An immutable code "commit"; push creates one pending, merge/rollback repoint active.
export interface AutomationVersion {
  id: string;
  automation_id: string;
  sequence: number;
  code: string;
  pusher_id: string;
  message?: string;
  status: AutomationVersionStatus;
  created_at: string;
}

// Both sides of the diff; pending is nil when there's nothing pending.
export interface AutomationVersionDiff {
  active: AutomationVersion | null;
  pending: AutomationVersion | null;
}
