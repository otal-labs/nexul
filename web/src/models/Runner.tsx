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
  // What the viewer holds on it: machines:see, machines:deploy, machines:write (Manage) and machines:delete.
  actions: z.array(z.string()),
});

export type Machine = z.infer<typeof MachineSchema>;

export const QueuedJobSchema = z.object({
  id: z.string().min(1),
  kind: z.string(),
  service: z.string().optional(),
  // The machine the job must run on; empty when any machine's runner may take it.
  target: z.string().optional(),
});

export type QueuedJob = z.infer<typeof QueuedJobSchema>;

// Names become service names on the machine, so they share the installer's pattern.
export const RunnerEnrollmentFormSchema = z.object({
  name: z
    .string()
    .trim()
    .regex(/^[a-z0-9][a-z0-9-]{0,31}$/, "Use 1 to 32 lowercase letters, digits or dashes"),
});

export type RunnerEnrollmentFormData = z.infer<typeof RunnerEnrollmentFormSchema>;

// Set when the runner joins a known machine; otherwise the server names the machine after the computer's hostname.
export interface RunnerEnrollmentInput extends RunnerEnrollmentFormData {
  machine?: string | undefined;
}

export const RunnerEnrollmentSchema = z.object({
  code: z.string(),
  expires_at: z.string(),
  commands: z.object({ unix: z.string(), windows: z.string() }),
});

export type RunnerEnrollment = z.infer<typeof RunnerEnrollmentSchema>;

// An unused code its maker may still revoke; never the code itself. A computer's carries computer_id.
export interface OpenEnrollment {
  id: string;
  name: string;
  machine?: string;
  computer_id?: string;
  created_at: string;
  expires_at: string;
}
