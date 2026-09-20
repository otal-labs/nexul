import { z } from "zod";

import { VersionLatestSchema } from "@/models/Version";

export const UpgradeRecordStatus = {
  Pending: "pending",
  Started: "started",
  Completed: "completed",
  Failed: "failed",
} as const;

export type UpgradeRecordStatus = (typeof UpgradeRecordStatus)[keyof typeof UpgradeRecordStatus];

export const InstanceUpgradeRecordSchema = z.object({
  id: z.string(),
  from_version: z.string(),
  to_version: z.string(),
  status: z.enum([
    UpgradeRecordStatus.Pending,
    UpgradeRecordStatus.Started,
    UpgradeRecordStatus.Completed,
    UpgradeRecordStatus.Failed,
  ]),
  error: z.string(),
  requested_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type InstanceUpgradeRecord = z.infer<typeof InstanceUpgradeRecordSchema>;

export const InstanceUpgradeSchema = z.object({
  version: z.string(),
  channel: z.string(),
  latest: VersionLatestSchema.nullable(),
  update_available: z.boolean(),
  can_upgrade: z.boolean(),
  reason: z.string(),
  upgrade: InstanceUpgradeRecordSchema.nullable(),
});

export type InstanceUpgrade = z.infer<typeof InstanceUpgradeSchema>;

// Both `pending` (record written) and `started` (helper running) mean the section polls and hides the button.
export const isUpgradeInProgress = (record: InstanceUpgradeRecord | null): boolean =>
  record?.status === UpgradeRecordStatus.Pending || record?.status === UpgradeRecordStatus.Started;
