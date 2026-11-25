// The Agent page's open record: a play, a computer (yours or shared), or a project's run settings ("defaults" for every
// project's), each a workspace-relative path (ADR 0211).
export type AgentKind = "plays" | "computers" | "projects";

export const AGENT_PATH = "/agent";

export const DEFAULTS_ID = "defaults";

export const agentPath = (kind: AgentKind, id: string) => `${AGENT_PATH}/${kind}/${encodeURIComponent(id)}`;

export const isAgentKind = (value: string | undefined): value is AgentKind => value === "plays" || value === "computers" || value === "projects";
