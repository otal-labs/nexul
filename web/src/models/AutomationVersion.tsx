import type { AutomationVersionStatus } from "@/enums/Automation";

// An immutable code "commit"; push creates one pending, merge/rollback repoint active.
export interface AutomationVersion {
  id: string;
  automation_id: string;
  sequence: number;
  code: string;
  // The file the bundle in code was built from; absent for versions pushed before sources were kept.
  source?: string;
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

// What the diff compares: the sources when both sides kept one, else the bundles, which inline the SDK.
export const diffSides = ({ active, pending }: AutomationVersionDiff) => {
  const bySource = !!pending?.source && (!active || !!active.source);
  if (bySource) return { bySource, before: active?.source ?? "", after: pending.source ?? "" };
  return { bySource, before: active?.code ?? "", after: pending?.code ?? "" };
};
