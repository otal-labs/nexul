// Mirrors web/src/models/Access.tsx: the permission each gated area needs, so the phone and the browser agree.
export const AREA_PERMISSION = {
  runners: "runners:read",
  topology: "topology:read",
  automations: "automations:read",
  projects: "projects:read",
  tickets: "tickets:read",
  memories: "memories:read",
  docs: "docs:read",
  stacks: "stacks:read",
  stackLogs: "stacks:logs",
  deploys: "deploys:read",
  dns: "dns:read",
  newConversation: "chat:write",
  newProject: "projects:write",
  newDoc: "docs:write",
} as const;

export type Area = keyof typeof AREA_PERMISSION;
