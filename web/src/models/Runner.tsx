import { z } from "zod";

export const RequestKind = {
  Build: "build",
  Deploy: "deploy",
} as const;

export type RequestKind = (typeof RequestKind)[keyof typeof RequestKind];

export const RunningJobSchema = z.object({
  id: z.string(),
  kind: z.string(),
  service: z.string().optional(),
});

export type RunningJob = z.infer<typeof RunningJobSchema>;

export const RunnerSchema = z.object({
  id: z.string().min(1),
  name: z.string(),
  connected: z.boolean(),
  last_seen: z.string(),
  running_job: RunningJobSchema.nullable(),
  version: z.string(),
  // The machine this runner belongs to (issue 05); empty until its first connect resolves one.
  machine: z.string().optional(),
});

export type Runner = z.infer<typeof RunnerSchema>;

// A host runners connect from; runners reporting the same machine form a dispatch pool.
export const MachineSchema = z.object({
  id: z.string().min(1),
  name: z.string(),
  stack_root: z.string(),
  reported_hostname: z.string().optional(),
  first_seen: z.string(),
  last_seen: z.string(),
});

export type Machine = z.infer<typeof MachineSchema>;

export const QueuedJobSchema = z.object({
  id: z.string().min(1),
  kind: z.string(),
  service: z.string().optional(),
});

export type QueuedJob = z.infer<typeof QueuedJobSchema>;

export const RunnerInstallSchema = z.object({
  ws_url: z.string(),
  secret: z.string(),
  download_url: z.string(),
});

export type RunnerInstall = z.infer<typeof RunnerInstallSchema>;
