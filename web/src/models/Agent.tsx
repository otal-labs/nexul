// The Agent page's open record, as a workspace-relative path (ADR 0211); "defaults" is every project's run settings.
export type AgentKind = "plays" | "computers" | "projects";

export const AGENT_PATH = "/agent";

export const DEFAULTS_ID = "defaults";

export const agentPath = (kind: AgentKind, id: string) => `${AGENT_PATH}/${kind}/${encodeURIComponent(id)}`;

export const isAgentKind = (value: string | undefined): value is AgentKind => value === "plays" || value === "computers" || value === "projects";
