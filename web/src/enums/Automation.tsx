// Mirrors internal/automations/model.go's Kind.
export const AutomationKind = {
  Default: "default",
  Custom: "custom",
} as const;

export type AutomationKind = (typeof AutomationKind)[keyof typeof AutomationKind];

// Mirrors internal/automations/versions.go's VersionStatus.
export const AutomationVersionStatus = {
  Pending: "pending",
  Active: "active",
  Inactive: "inactive",
} as const;

export type AutomationVersionStatus = (typeof AutomationVersionStatus)[keyof typeof AutomationVersionStatus];

// Mirrors internal/automations/runs.go's RunOutcome.
export const AutomationRunOutcome = {
  Success: "success",
  Failure: "failure",
  Crash: "crash",
} as const;

export type AutomationRunOutcome = (typeof AutomationRunOutcome)[keyof typeof AutomationRunOutcome];
